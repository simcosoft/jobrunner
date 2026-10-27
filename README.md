# JobRunner

JobRunner is a framework for performing work asynchronously. It comes with cron to schedule and queue job functions for processing at a specified time.

## Install

`go get github.com/simcosoft/jobrunner`

### Setup

```go
package main

import "github.com/simcosoft/jobrunner"

func main() {
    jobrunner.Start() // optional: jobrunner.Start(pool int, concurrent int) (10, 1)
    jobrunner.Schedule("@every 5s", ReminderEmails{})
}

// Job Specific Functions
type ReminderEmails struct {
    // filtered
}

// ReminderEmails.Run() will get triggered automatically.
func (e ReminderEmails) Run() {
    // Queries the DB
    // Sends some email
    fmt.Printf("Every 5 sec send reminder emails \n")
}
```

## Use cases
Here are some examples of what we use JobRunner for:

* Send emails to new users after signup
* Sending push notification or emails based on specifics
* ReMarketing Engine - send invites, reminder emails, etc ...
* Clean DB, data or AMZ S3
* Sending Server stats to monitoring apps
* Send data stats at daily or weekly intervals

### Supported Features
*All jobs are processed outside of the request flow*

* Now: process a job immediately
* In: processing a job one time, after a given time
* Every: process a recurring job after every given time gap
* Schedule: process a job (recurring or otherwise) at a given time

## Basics

```go
    jobrunner.Schedule("* */5 * * * *", DoSomething{}) // every 5min do something
    jobrunner.Schedule("@every 1h30m10s", ReminderEmails{})
    jobrunner.Schedule("@midnight", DataStats{}) // every midnight do this..
    jobrunner.Every(16*time.Minute, CleanS3{}) // evey 16 min clean...
    jobrunner.In(10*time.Second, WelcomeEmail{}) // one time job. starts after 10sec
    jobrunner.Now(NowDo{}) // do the job as soon as it's triggered
```
[**More Detailed CRON Specs**](https://github.com/robfig/cron/blob/v2/doc.go)

## Contribute

**Use issues for everything**

- Report problems
- Discuss before sending pull request
- Suggest new features
- Improve/fix documentation


#### License
MIT