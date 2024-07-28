# LazyWatch (WIP!)

A simple lazy filesystem watcher for web servers. LazyWatch will watch the filesystem of your project and keep track if any changes have been made. If so, it will re-run your start command, but only when a request has been made to your project's server.

## Example Usage

Once installed, run `lazywatch -p 3001:3000 -c "go run main.go"`. 

This will proxy traffic from port 3001 to 3000 (where your webserver project would be configured to listen). LazyWatch will run the specified command (`go run main.go` in this case) and then watch the project's filesystem. On a modified or created file, it will "invalidate" the current command. When another request is made to the webserver, it will first kill the running process and start a fresh one. It will wait until that webserver is healthy until it starts forwarding the traffic again. 
