/*
Command jf is a JSON flattener, which composes well with existing system tools, e.g., with grep and diff; jf transforms a JSON element into a sequence of path-value pairs, one per row, tab-separated.

	; echo '{"fruit":[{"name":"banana"},{"name":"apple"}]}' | jf | tab
	.                  {}
	."fruit"           []
	."fruit"[0]        {}
	."fruit"[0]."name" "banana"
	."fruit"[1]        {}
	."fruit"[1]."name" "apple"

There's a (simpler and faster) version in C (for Plan 9 and p9p) in the 9 subdirectory. It's easier to install the Go version, though.

Motivation: I wanted a simple and composable tool that would take advantage of other software already in the system (following the UNIX philosophy), for exploring and comparing JSON documents. Also, as a learning exercise in lexing/parsing.

Flattening a single element:

	; curl -sL 'https://api.github.com/search/users?q=nicola@aloc.in' | jf | tab
	.                                 {}
	."total_count"                    1
	."incomplete_results"             false
	."items"                          []
	."items"[0]                       {}
	."items"[0]."login"               "nicolagi"
	."items"[0]."id"                  1922900
	."items"[0]."node_id"             "MDQ6VXNlcjE5MjI5MDA="
	."items"[0]."avatar_url"          "https://avatars1.githubusercontent.com/u/1922900?v=4"
	."items"[0]."gravatar_id"         ""
	."items"[0]."url"                 "https://api.github.com/users/nicolagi"
	."items"[0]."html_url"            "https://github.com/nicolagi"
	."items"[0]."followers_url"       "https://api.github.com/users/nicolagi/followers"
	."items"[0]."following_url"       "https://api.github.com/users/nicolagi/following{/other_user}"
	."items"[0]."gists_url"           "https://api.github.com/users/nicolagi/gists{/gist_id}"
	."items"[0]."starred_url"         "https://api.github.com/users/nicolagi/starred{/owner}{/repo}"
	."items"[0]."subscriptions_url"   "https://api.github.com/users/nicolagi/subscriptions"
	."items"[0]."organizations_url"   "https://api.github.com/users/nicolagi/orgs"
	."items"[0]."repos_url"           "https://api.github.com/users/nicolagi/repos"
	."items"[0]."events_url"          "https://api.github.com/users/nicolagi/events{/privacy}"
	."items"[0]."received_events_url" "https://api.github.com/users/nicolagi/received_events"
	."items"[0]."type"                "User"
	."items"[0]."site_admin"          false
	."items"[0]."score"               1.0

Compare two JSON documents by composing jf with the system diff:

	:; diff -u <(jf < before.json) <(jf < after.json)

A concrete example:

	:; function prep() { curl -sL $1 | jf | tab; }
	:; diff -u <(prep https://api.spacexdata.com/v3/capsules/C101) <(prep https://api.spacexdata.com/v3/capsules/C102)
	@@ -1,14 +1,14 @@
	 .                       {}
	-."capsule_serial"       "C101"
	+."capsule_serial"       "C102"
	 ."capsule_id"           "dragon1"
	 ."status"               "retired"
	-."original_launch"      "2010-12-08T15:43:00.000Z"
	-."original_launch_unix" 1291822980
	+."original_launch"      "2012-05-22T07:44:00.000Z"
	+."original_launch_unix" 1335944640
	 ."missions"             []
	 ."missions"[0]          {}
	-."missions"[0]."name"   "COTS 1"
	-."missions"[0]."flight" 7
	+."missions"[0]."name"   "COTS 2"
	+."missions"[0]."flight" 8
	 ."landings"             1
	 ."type"                 "Dragon 1.0"
	-."details"              "Reentered after three weeks in orbit"
	+."details"              "First Dragon spacecraft"
	 ."reuse_count"          0

Have object keys been reordered from one document to the other? No need to add features to jf, just bring sort to the mix:

	diff -u (sort <before.json | jf) <(sort <after.json | jf)

Have array items been reordered from one document to the other? No need to add features to jf, just bring sed to the mix:

	function prep() {
		cat $1 | sed -E 's/\[[0-9]+\]/[number]/g' | sort
	}
	diff -u <(prep before.json) <(prep after.json)

Want extract all SpaceX launches videos? Post-process jf's output with grep and awk.

	; curl -sL https://api.spacexdata.com/v3/launches | jf | grep video_link | awk '{print $2}' | sed 3q
	"https://www.youtube.com/watch?v=0a_00nJ_Y88"
	"https://www.youtube.com/watch?v=Lk4zQ2wP-Nc"
	"https://www.youtube.com/watch?v=v0w9p3U8860"

By default only one element is accepted:

	; { curl -sL https://api.spacexdata.com/v3/capsules/C101 ; curl -sL https://api.spacexdata.com/v3/capsules/C102 } | jf
	2020/06/01 18:56:21 main: expected to flatten one value and get EOF, got: {
	<output for first JSON element>

With -m, jf will flatten many JSON elements but paths may be duplicated as a result:

	; { curl -sL https://api.spacexdata.com/v3/capsules/C101 ; curl -sL https://api.spacexdata.com/v3/capsules/C102 } | jf -m | sort | tab
	.                       {}
	.                       {}
	."capsule_id"           "dragon1"
	."capsule_id"           "dragon1"
	."capsule_serial"       "C101"
	."capsule_serial"       "C102"
	."details"              "First Dragon spacecraft"
	."details"              "Reentered after three weeks in orbit"
	."landings"             1
	."landings"             1
	."missions"             []
	."missions"             []
	."missions"[0]          {}
	."missions"[0]          {}
	."missions"[0]."flight" 7
	."missions"[0]."flight" 8
	."missions"[0]."name"   "COTS 1"
	."missions"[0]."name"   "COTS 2"
	."original_launch"      "2010-12-08T15:43:00.000Z"
	."original_launch"      "2012-05-22T07:44:00.000Z"
	."original_launch_unix" 1291822980
	."original_launch_unix" 1335944640
	."reuse_count"          0
	."reuse_count"          0
	."status"               "retired"
	."status"               "retired"
	."type"                 "Dragon 1.0"
	."type"                 "Dragon 1.0"

The input to jf is a stream, and the output is incremental, so that jf works
well in a pipeline. For example, in a pipeline like

	cat haystack.json | jf | grep needle

all three processes make progress at the same time. In other words,
it's not necessary to parse the whole document in order to spit out
the path-value pairs. An extreme example to make this point clearer:
Consider a pipeline to print just the first two lines of jf's or jq's
output. With jf it's instantaneous:

	:; time cat citylots.json| jf | sed 2q
	.       {}
	."type" "FeatureCollection"
	cat citylots.json  0.00s user 0.00s system 39% cpu 0.006 total
	jf  0.00s user 0.01s system 103% cpu 0.005 total
	sed 2q  0.00s user 0.00s system 87% cpu 0.004 total

With jq the third process, sed, can't run until jq is done. It's not
a real pipeline, it's a sequence in practice:

	:; time cat citylots.json | jq . | sed 2q
	{
	  "type": "FeatureCollection",
	cat citylots.json  0.03s user 0.35s system 5% cpu 6.614 total
	jq .  6.06s user 0.60s system 99% cpu 6.670 total
	sed 2q  0.00s user 0.00s system 0% cpu 6.615 total

*/
package main
