# CSC-482 GoServer
- This Repository holds the code for a GoServer that will run in a Docker Container on an Amazon EC2 instance
- The server will listen for requests on the "jwilcox5/status" endpoint on the :35000 port. When a request to that endpoint is made, a JSON object containing the system time and a HTTP Status of 200 is returned
- If an invalid endpoint is returned a HTTP Status of 404 is returned and if a request that is not a "GET" request is made, a HTTP Status of 405 is returned
- A record of each request, good or bad, will also be sent to Loggly containing the method type, source IP address, request path, and the HTTP Status code
