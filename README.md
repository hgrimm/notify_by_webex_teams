notify_by_webex_teams
=====================
CLI command for sending messages to Cisco Webex Teams rooms.
Send messages and files to Webex Teams. If *room name* is not found a new room is created.
Supports on file upload per request. 
by Herwig Grimm (herwig.grimm at aon.at)

required args
-------------
			-T <Webex Teams API token>
			-t <team name>
			-r <room name>
			-m <markdown message>

optional args
-------------
			-p <proxy server>
			-f <filename and path to send>

example
-------
			notify_by_webex_teams.exe -T <apitoken> -t "KMP-Team" -r "My New Room" -m "Happy hacking" -f logo.png

doc links
---------
			https:developer.webex.com/getting-started.html
