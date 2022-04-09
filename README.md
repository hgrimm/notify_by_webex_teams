notify_by_webex_teams
=====================
CLI command for sending messages to Cisco Webex rooms or Cisco Webex recipient.
Send messages and files to Webex Teams. If *room name* is not found a new room is created.
Supports on file upload per request. 
by Herwig Grimm (herwig.grimm at aon.at)

required args
-------------
```
-T <Webex Teams API token>
-t <team name>
-r <room name>
-m <markdown message>
```
or

```
-T <Webex Teams API token>
-D <email address>
-m <markdown message>
```

optional flags
--------------
```
-p <proxy server>
-f <filename and path to send>
-a <card attachment>
-i 

```

flag details:
-------------
    a ... card attachment -a see https://developer.webex.com/docs/api/guides/cards and https://adaptivecards.io/designer/
    d ... delete message. provide message id
    D ... Webex email address of the recipient when sending a private 1:1 message
    f ... PNG filename and path to send
    i ... read message from standard input
    m ... markdown message
    p ... proxy server. format: http://<user>:<password>@<hostname>:<port>
    r ... Webex room name
    T ... Webex bot token (bot must be member of team and room)")
    t ... Webex team name
    V ... show version
    



example
-------
```
notify_by_webex_teams.exe -T <apitoken> -t "KMP-Team" -r "My New Room" -m "Happy hacking" -f logo.png
notify_by_webex_teams.exe -T <apitoken> -D john.smith@example.com -m "A direct message." -f logo.png
```

doc links
---------

[Cisco Webex Developer - Bot Section](https://developer.webex.com/docs/bots)

