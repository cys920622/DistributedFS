Special instructions for compiling/running the code should be included in this file.

>Server-side logging:
./server [-log] [server-address]

or

go run server.go [-log] [server-address:port]

Run with the -log flag to output server-side logs to the console. The server runs
silently when this flag is not included.


>Client-side logging:
For debugging purposes only.
'const LoggingOn' can be flipped to 'true'  in the code to output client-side
logging to the console. Should normally be turned off.


>Running unit tests:
Unit tests can be run with app.go [server-address:port].
