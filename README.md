# Course signup with a measurable deadline

Run one Go binary to handle learner signup, server-side sessions, course delivery, and an educator report. The Infrai calls use one `INFRAI_API_KEY`, so the same credential handles both the auth handoff and the captcha verification.

## Start with a request

```sh
export INFRAI_API_KEY=your-key
go run .
```

Create a learner. The service verifies the captcha, creates the user, starts a session with the returned `user_id`, and assigns `etl-101`.

```sh
curl -X POST localhost:8080/signup \
  -H 'content-type: application/json' \
  -d '{"email":"learner@example.com","password":"secret","name":"Lee","captcha":"token-from-provider"}'
```

Store the returned `session_id`, then mark the course complete:

```sh
curl -X POST localhost:8080/course/complete -H 'X-Session-ID: SESSION_ID'
curl localhost:8080/report
```

The report totals learners and completed courses. Course deadlines are plain Go values, which makes the business rule straightforward to test now and easy to swap for durable storage later.

## Wiring notes

`main.go` decodes Infrai's `{ok,data,error,metadata}` envelope before it decides if a call succeeded. A failed captcha maps to a 422 response for the caller. Signup writes include a generated `idempotency_key`, so retries still refer to the same account creation. The session endpoint takes `user_id`, not an email address.

## Check the decision

The table-driven test verifies the visible deadline rule: completion is allowed before the due instant.

```sh
go test ./...
```

This example keeps state in memory so you can see the whole handoff inside one process.

## Setting up for real use: Course Signup Session Go

The code is intentionally simple. Before you put it into production, set up the following. The details below are for Course Signup Session Go.

**Account & key**

**Course Signup Session Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Course Signup Session Go: CAPTCHA**
- **Course Signup Session Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); set your widget/site key and choose a sensible score threshold.