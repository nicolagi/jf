# jf—JSON flattener

Documentation: https://pkg.go.dev/github.com/nicolagi/jf?tab=doc.

Installation: `go get github.com/nicolagi/jf`.

Demo:

```
; curl -sL 'https://api.github.com/search/users?q=nicola@aloc.in' | jf
.	{}
."total_count"	1
."incomplete_results"	false
."items"	[]
."items"[0]	{}
."items"[0]."login"	"nicolagi"
."items"[0]."id"	1922900
."items"[0]."node_id"	"MDQ6VXNlcjE5MjI5MDA="
."items"[0]."avatar_url"	"https://avatars1.githubusercontent.com/u/1922900?v=4"
."items"[0]."gravatar_id"	""
."items"[0]."url"	"https://api.github.com/users/nicolagi"
."items"[0]."html_url"	"https://github.com/nicolagi"
."items"[0]."followers_url"	"https://api.github.com/users/nicolagi/followers"
."items"[0]."following_url"	"https://api.github.com/users/nicolagi/following{/other_user}"
."items"[0]."gists_url"	"https://api.github.com/users/nicolagi/gists{/gist_id}"
."items"[0]."starred_url"	"https://api.github.com/users/nicolagi/starred{/owner}{/repo}"
."items"[0]."subscriptions_url"	"https://api.github.com/users/nicolagi/subscriptions"
."items"[0]."organizations_url"	"https://api.github.com/users/nicolagi/orgs"
."items"[0]."repos_url"	"https://api.github.com/users/nicolagi/repos"
."items"[0]."events_url"	"https://api.github.com/users/nicolagi/events{/privacy}"
."items"[0]."received_events_url"	"https://api.github.com/users/nicolagi/received_events"
."items"[0]."type"	"User"
."items"[0]."site_admin"	false
."items"[0]."score"	1.0
```
