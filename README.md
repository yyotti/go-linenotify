# go-linenotify
Helper functions for sending message by LINE Notify API.

## Install
```sh
go get github.com/yyotti/go-linenotify
```

## Example
```go
package main

import (
	"context"
	"log"

	"github.com/yyotti/go-linenotify"
)

const authToken = "[LINE Notify Authorization Token]"

func main() {
	notifier, err := linenotify.New(authToken)
	if err != nil {
		log.Fatal(err)
	}

	err = notifier.Send(context.Background(), "Hello!")
	if err != nil {
		log.Fatal(err)
	}
}
```
