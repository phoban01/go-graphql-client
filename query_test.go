package graphql

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

type cachedDirective struct {
	ttl int
}

func (cd cachedDirective) Type() OptionType {
	return OptionTypeOperationDirective
}

func (cd cachedDirective) String() string {
	if cd.ttl <= 0 {
		return "@cached"
	}
	return fmt.Sprintf("@cached(ttl: %d)", cd.ttl)
}

func TestConstructQuery(t *testing.T) {
	tests := []struct {
		options     []Option
		inV         interface{}
		inVariables map[string]interface{}
		want        string
	}{
		{
			inV: struct {
				Viewer struct {
					Login      String
					CreatedAt  DateTime
					ID         ID
					DatabaseID Int
				}
				RateLimit struct {
					Cost      Int
					Limit     Int
					Remaining Int
					ResetAt   DateTime
				}
			}{},
			want: `{viewer{login,createdAt,id,databaseId},rateLimit{cost,limit,remaining,resetAt}}`,
		},
		{
			options: []Option{OperationName("GetRepository"), cachedDirective{}},
			inV: struct {
				Repository struct {
					DatabaseID Int
					URL        URI

					Issue struct {
						Comments struct {
							Edges []struct {
								Node struct {
									Body   String
									Author struct {
										Login String
									}
									Editor struct {
										Login String
									}
								}
								Cursor String
							}
						} `graphql:"comments(first:1after:\"Y3Vyc29yOjE5NTE4NDI1Ng==\")"`
					} `graphql:"issue(number:1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `query GetRepository @cached {repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1after:"Y3Vyc29yOjE5NTE4NDI1Ng=="){edges{node{body,author{login},editor{login}},cursor}}}}}`,
		},
		{
			inV: func() interface{} {
				type actor struct {
					Login     String
					AvatarURL URI
					URL       URI
				}

				return struct {
					Repository struct {
						DatabaseID Int
						URL        URI

						Issue struct {
							Comments struct {
								Edges []struct {
									Node struct {
										DatabaseID      Int
										Author          actor
										PublishedAt     DateTime
										LastEditedAt    *DateTime
										Editor          *actor
										Body            String
										ViewerCanUpdate Boolean
									}
									Cursor String
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `{repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1){edges{node{databaseId,author{login,avatarUrl,url},publishedAt,lastEditedAt,editor{login,avatarUrl,url},body,viewerCanUpdate},cursor}}}}}`,
		},
		{
			inV: func() interface{} {
				type actor struct {
					Login     String
					AvatarURL URI `graphql:"avatarUrl(size:72)"`
					URL       URI
				}

				return struct {
					Repository struct {
						Issue struct {
							Author         actor
							PublishedAt    DateTime
							LastEditedAt   *DateTime
							Editor         *actor
							Body           String
							ReactionGroups []struct {
								Content ReactionContent
								Users   struct {
									TotalCount Int
								}
								ViewerHasReacted Boolean
							}
							ViewerCanUpdate Boolean

							Comments struct {
								Nodes []struct {
									DatabaseID     Int
									Author         actor
									PublishedAt    DateTime
									LastEditedAt   *DateTime
									Editor         *actor
									Body           String
									ReactionGroups []struct {
										Content ReactionContent
										Users   struct {
											TotalCount Int
										}
										ViewerHasReacted Boolean
									}
									ViewerCanUpdate Boolean
								}
								PageInfo struct {
									EndCursor   String
									HasNextPage Boolean
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `{repository(owner:"shurcooL-test"name:"test-repo"){issue(number:1){author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate,comments(first:1){nodes{databaseId,author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate},pageInfo{endCursor,hasNextPage}}}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body String
					} `graphql:"issue(number: 1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `{repository(owner:"shurcooL-test"name:"test-repo"){issue(number: 1){body}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body String
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
			}{},
			inVariables: map[string]interface{}{
				"repositoryOwner": String("shurcooL-test"),
				"repositoryName":  String("test-repo"),
				"issueNumber":     Int(1),
			},
			want: `query ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){body}}}`,
		},
		{
			options: []Option{OperationName("SearchRepository"), cachedDirective{100}},
			inV: struct {
				Repository struct {
					Issue struct {
						ReactionGroups []struct {
							Users struct {
								Nodes []struct {
									Login String
								}
							} `graphql:"users(first:10)"`
						}
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
			}{},
			inVariables: map[string]interface{}{
				"repositoryOwner": String("shurcooL-test"),
				"repositoryName":  String("test-repo"),
				"issueNumber":     Int(1),
			},
			want: `query SearchRepository($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!) @cached(ttl: 100) {repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// check test above works with repository inner map
		{
			inV: func() interface{} {
				type query struct {
					Repository [][2]interface{} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
				}
				type issue struct {
					ReactionGroups []struct {
						Users struct {
							Nodes []struct {
								Login String
							}
						} `graphql:"users(first:10)"`
					}
				}
				return query{Repository: [][2]interface{}{
					{"issue(number: $issueNumber)", issue{}},
				}}
			}(),
			inVariables: map[string]interface{}{
				"repositoryOwner": String("shurcooL-test"),
				"repositoryName":  String("test-repo"),
				"issueNumber":     Int(1),
			},
			want: `query ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// check inner maps work inside slices
		{
			inV: func() interface{} {
				type query struct {
					Repository [][2]interface{} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
				}
				type issue struct {
					ReactionGroups []struct {
						Users [][2]interface{} `graphql:"users(first:10)"`
					}
				}
				type nodes []struct {
					Login String
				}
				return query{Repository: [][2]interface{}{
					{"issue(number: $issueNumber)", issue{
						ReactionGroups: []struct {
							Users [][2]interface{} `graphql:"users(first:10)"`
						}{
							{Users: [][2]interface{}{
								{"nodes", nodes{}},
							}},
						},
					}},
				}}
			}(),
			inVariables: map[string]interface{}{
				"repositoryOwner": String("shurcooL-test"),
				"repositoryName":  String("test-repo"),
				"issueNumber":     Int(1),
			},
			want: `query ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// Embedded structs without graphql tag should be inlined in query.
		{
			inV: func() interface{} {
				type actor struct {
					Login     String
					AvatarURL URI
					URL       URI
				}
				type event struct { // Common fields for all events.
					Actor     actor
					CreatedAt DateTime
				}
				type IssueComment struct {
					Body String
				}
				return struct {
					event                                         // Should be inlined.
					IssueComment  `graphql:"... on IssueComment"` // Should not be, because of graphql tag.
					CurrentTitle  String
					PreviousTitle String
					Label         struct {
						Name  String
						Color String
					}
				}{}
			}(),
			want: `{actor{login,avatarUrl,url},createdAt,... on IssueComment{body},currentTitle,previousTitle,label{name,color}}`,
		},
		{
			inV: struct {
				Viewer struct {
					Login      string
					CreatedAt  time.Time
					ID         interface{}
					DatabaseID int
				}
			}{},
			want: `{viewer{login,createdAt,id,databaseId}}`,
		},
		{
			inV: struct {
				Viewer struct {
					ID         interface{}
					Login      string
					CreatedAt  time.Time
					DatabaseID int
				}
				Tags map[string]interface{} `scalar:"true"`
			}{},
			want: `{viewer{id,login,createdAt,databaseId},tags}`,
		},
		{
			inV: struct {
				Viewer struct {
					ID         interface{}
					Login      string
					CreatedAt  time.Time
					DatabaseID int
				} `scalar:"true"`
			}{},
			want: `{viewer}`,
		},
		{
			inV: struct {
				Viewer struct {
					ID         interface{} `graphql:"-"`
					Login      string
					CreatedAt  time.Time `graphql:"-"`
					DatabaseID int
				}
			}{},
			want: `{viewer{login,databaseId}}`,
		},
	}
	for _, tc := range tests {
		got, err := ConstructQuery(tc.inV, tc.inVariables, tc.options...)
		if err != nil {
			t.Error(err)
		} else if got != tc.want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, tc.want)
		}
	}
}

type CreateUser struct {
	Login string
}

type DeleteUser struct {
	Login string
}

func TestConstructMutation(t *testing.T) {
	tests := []struct {
		inV         interface{}
		inVariables map[string]interface{}
		want        string
	}{
		{
			inV: struct {
				AddReaction struct {
					Subject struct {
						ReactionGroups []struct {
							Users struct {
								TotalCount Int
							}
						}
					}
				} `graphql:"addReaction(input:$input)"`
			}{},
			inVariables: map[string]interface{}{
				"input": AddReactionInput{
					SubjectID: "MDU6SXNzdWUyMzE1MjcyNzk=",
					Content:   ReactionContentThumbsUp,
				},
			},
			want: `mutation ($input:AddReactionInput!){addReaction(input:$input){subject{reactionGroups{users{totalCount}}}}}`,
		},
		{
			inV: [][2]interface{}{
				{"createUser(login:$login1)", &CreateUser{}},
				{"deleteUser(login:$login2)", &DeleteUser{}},
			},
			inVariables: map[string]interface{}{
				"login1": String("grihabor"),
				"login2": String("diman"),
			},
			want: "mutation ($login1:String!$login2:String!){createUser(login:$login1){login}deleteUser(login:$login2){login}}",
		},
	}
	for _, tc := range tests {
		got, err := ConstructMutation(tc.inV, tc.inVariables)
		if err != nil {
			t.Error(err)
		} else if got != tc.want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, tc.want)
		}
	}
}

func TestConstructSubscription(t *testing.T) {
	tests := []struct {
		name        string
		inV         interface{}
		inVariables map[string]interface{}
		want        string
	}{
		{
			inV: struct {
				Viewer struct {
					Login      String
					CreatedAt  DateTime
					ID         ID
					DatabaseID Int
				}
				RateLimit struct {
					Cost      Int
					Limit     Int
					Remaining Int
					ResetAt   DateTime
				}
			}{},
			want: `subscription{viewer{login,createdAt,id,databaseId},rateLimit{cost,limit,remaining,resetAt}}`,
		},
		{
			name: "GetRepository",
			inV: struct {
				Repository struct {
					DatabaseID Int
					URL        URI

					Issue struct {
						Comments struct {
							Edges []struct {
								Node struct {
									Body   String
									Author struct {
										Login String
									}
									Editor struct {
										Login String
									}
								}
								Cursor String
							}
						} `graphql:"comments(first:1after:\"Y3Vyc29yOjE5NTE4NDI1Ng==\")"`
					} `graphql:"issue(number:1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `subscription GetRepository{repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1after:"Y3Vyc29yOjE5NTE4NDI1Ng=="){edges{node{body,author{login},editor{login}},cursor}}}}}`,
		},
		{
			inV: func() interface{} {
				type actor struct {
					Login     String
					AvatarURL URI
					URL       URI
				}

				return struct {
					Repository struct {
						DatabaseID Int
						URL        URI

						Issue struct {
							Comments struct {
								Edges []struct {
									Node struct {
										DatabaseID      Int
										Author          actor
										PublishedAt     DateTime
										LastEditedAt    *DateTime
										Editor          *actor
										Body            String
										ViewerCanUpdate Boolean
									}
									Cursor String
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `subscription{repository(owner:"shurcooL-test"name:"test-repo"){databaseId,url,issue(number:1){comments(first:1){edges{node{databaseId,author{login,avatarUrl,url},publishedAt,lastEditedAt,editor{login,avatarUrl,url},body,viewerCanUpdate},cursor}}}}}`,
		},
		{
			inV: func() interface{} {
				type actor struct {
					Login     String
					AvatarURL URI `graphql:"avatarUrl(size:72)"`
					URL       URI
				}

				return struct {
					Repository struct {
						Issue struct {
							Author         actor
							PublishedAt    DateTime
							LastEditedAt   *DateTime
							Editor         *actor
							Body           String
							ReactionGroups []struct {
								Content ReactionContent
								Users   struct {
									TotalCount Int
								}
								ViewerHasReacted Boolean
							}
							ViewerCanUpdate Boolean

							Comments struct {
								Nodes []struct {
									DatabaseID     Int
									Author         actor
									PublishedAt    DateTime
									LastEditedAt   *DateTime
									Editor         *actor
									Body           String
									ReactionGroups []struct {
										Content ReactionContent
										Users   struct {
											TotalCount Int
										}
										ViewerHasReacted Boolean
									}
									ViewerCanUpdate Boolean
								}
								PageInfo struct {
									EndCursor   String
									HasNextPage Boolean
								}
							} `graphql:"comments(first:1)"`
						} `graphql:"issue(number:1)"`
					} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
				}{}
			}(),
			want: `subscription{repository(owner:"shurcooL-test"name:"test-repo"){issue(number:1){author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate,comments(first:1){nodes{databaseId,author{login,avatarUrl(size:72),url},publishedAt,lastEditedAt,editor{login,avatarUrl(size:72),url},body,reactionGroups{content,users{totalCount},viewerHasReacted},viewerCanUpdate},pageInfo{endCursor,hasNextPage}}}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body String
					} `graphql:"issue(number: 1)"`
				} `graphql:"repository(owner:\"shurcooL-test\"name:\"test-repo\")"`
			}{},
			want: `subscription{repository(owner:"shurcooL-test"name:"test-repo"){issue(number: 1){body}}}`,
		},
		{
			inV: struct {
				Repository struct {
					Issue struct {
						Body String
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
			}{},
			inVariables: map[string]interface{}{
				"repositoryOwner": String("shurcooL-test"),
				"repositoryName":  String("test-repo"),
				"issueNumber":     Int(1),
			},
			want: `subscription ($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){body}}}`,
		},
		{
			name: "SearchRepository",
			inV: struct {
				Repository struct {
					Issue struct {
						ReactionGroups []struct {
							Users struct {
								Nodes []struct {
									Login String
								}
							} `graphql:"users(first:10)"`
						}
					} `graphql:"issue(number: $issueNumber)"`
				} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
			}{},
			inVariables: map[string]interface{}{
				"repositoryOwner": String("shurcooL-test"),
				"repositoryName":  String("test-repo"),
				"issueNumber":     Int(1),
			},
			want: `subscription SearchRepository($issueNumber:Int!$repositoryName:String!$repositoryOwner:String!){repository(owner: $repositoryOwner, name: $repositoryName){issue(number: $issueNumber){reactionGroups{users(first:10){nodes{login}}}}}}`,
		},
		// Embedded structs without graphql tag should be inlined in query.
		{
			inV: func() interface{} {
				type actor struct {
					Login     String
					AvatarURL URI
					URL       URI
				}
				type event struct { // Common fields for all events.
					Actor     actor
					CreatedAt DateTime
				}
				type IssueComment struct {
					Body String
				}
				return struct {
					event                                         // Should be inlined.
					IssueComment  `graphql:"... on IssueComment"` // Should not be, because of graphql tag.
					CurrentTitle  String
					PreviousTitle String
					Label         struct {
						Name  String
						Color String
					}
				}{}
			}(),
			want: `subscription{actor{login,avatarUrl,url},createdAt,... on IssueComment{body},currentTitle,previousTitle,label{name,color}}`,
		},
		{
			inV: struct {
				Viewer struct {
					Login      string
					CreatedAt  time.Time
					ID         interface{}
					DatabaseID int
				}
			}{},
			want: `subscription{viewer{login,createdAt,id,databaseId}}`,
		},
	}
	for _, tc := range tests {
		got, err := ConstructSubscription(tc.inV, tc.inVariables, OperationName(tc.name))
		if err != nil {
			t.Error(err)
		} else if got != tc.want {
			t.Errorf("\ngot:  %q\nwant: %q\n", got, tc.want)
		}
	}
}

func TestQueryArguments(t *testing.T) {
	tests := []struct {
		in   map[string]interface{}
		want string
	}{
		{
			in:   map[string]interface{}{"a": Int(123), "b": NewBoolean(true)},
			want: "$a:Int!$b:Boolean",
		},
		{
			in: map[string]interface{}{
				"required": []IssueState{IssueStateOpen, IssueStateClosed},
				"optional": &[]IssueState{IssueStateOpen, IssueStateClosed},
			},
			want: "$optional:[IssueState!]$required:[IssueState!]!",
		},
		{
			in: map[string]interface{}{
				"required": []IssueState(nil),
				"optional": (*[]IssueState)(nil),
			},
			want: "$optional:[IssueState!]$required:[IssueState!]!",
		},
		{
			in: map[string]interface{}{
				"required": [...]IssueState{IssueStateOpen, IssueStateClosed},
				"optional": &[...]IssueState{IssueStateOpen, IssueStateClosed},
			},
			want: "$optional:[IssueState!]$required:[IssueState!]!",
		},
		{
			in:   map[string]interface{}{"id": ID("someID")},
			want: "$id:ID!",
		},
		{
			in:   map[string]interface{}{"ids": []ID{"someID", "anotherID"}},
			want: `$ids:[ID!]!`,
		},
		{
			in:   map[string]interface{}{"ids": &[]ID{"someID", "anotherID"}},
			want: `$ids:[ID!]`,
		},
	}
	for i, tc := range tests {
		got := queryArguments(tc.in)
		if got != tc.want {
			t.Errorf("test case %d:\n got: %q\nwant: %q", i, got, tc.want)
		}
	}
}

// Custom GraphQL types for testing.
type (
	// DateTime is an ISO-8601 encoded UTC date.
	DateTime struct{ time.Time }

	// URI is an RFC 3986, RFC 3987, and RFC 6570 (level 4) compliant URI.
	URI struct{ *url.URL }
)

func (u *URI) UnmarshalJSON(data []byte) error { panic("mock implementation") }

// IssueState represents the possible states of an issue.
type IssueState string

// The possible states of an issue.
const (
	IssueStateOpen   IssueState = "OPEN"   // An issue that is still open.
	IssueStateClosed IssueState = "CLOSED" // An issue that has been closed.
)

// ReactionContent represents emojis that can be attached to Issues, Pull Requests and Comments.
type ReactionContent string

// Emojis that can be attached to Issues, Pull Requests and Comments.
const (
	ReactionContentThumbsUp   ReactionContent = "THUMBS_UP"   // Represents the 👍 emoji.
	ReactionContentThumbsDown ReactionContent = "THUMBS_DOWN" // Represents the 👎 emoji.
	ReactionContentLaugh      ReactionContent = "LAUGH"       // Represents the 😄 emoji.
	ReactionContentHooray     ReactionContent = "HOORAY"      // Represents the 🎉 emoji.
	ReactionContentConfused   ReactionContent = "CONFUSED"    // Represents the 😕 emoji.
	ReactionContentHeart      ReactionContent = "HEART"       // Represents the ❤️ emoji.
)

// AddReactionInput is an autogenerated input type of AddReaction.
type AddReactionInput struct {
	// The Node ID of the subject to modify. (Required.)
	SubjectID ID `json:"subjectId"`
	// The name of the emoji to react with. (Required.)
	Content ReactionContent `json:"content"`

	// A unique identifier for the client performing the mutation. (Optional.)
	ClientMutationID *String `json:"clientMutationId,omitempty"`
}
