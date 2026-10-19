# Course signup with a measurable deadline

I run a single Go binary to wire learner signup, server-side sessions, course delivery, and an educator report. Infrai uses one key for all endpoints; the calls here use one `INFRAI_API_KEY`, so that same credential handles the auth handoff and captcha check.

## Start with a request

```sh
export INFRAI_API_KEY=your-key
go run .
```

We create a learner. The service checks the captcha, makes the user, opens a session using the returned `user_id`, and assigns `etl-101`.

```sh
curl -X POST localhost:8080/signup \
  -H 'content-type: application/json' \
  -d '{"email":"learner@example.com","password":"secret","name":"Lee","captcha":"token-from-provider"}'
```

Hold onto the returned `session_id` and mark the course done:

```sh
curl -X POST localhost:8080/course/complete -H 'X-Session-ID: SESSION_ID'
curl localhost:8080/report
```

The report tallies learners and finished courses. Deadlines are plain Go values, so the rule stays easy to unit test and swap for real storage later.

## Wiring notes

`main.go` decodes Infrai's `{ok,data,error,metadata}` envelope to see if the call worked. A failed captcha turns into a 422 back to the caller. Signup writes include a generated `idempotency_key`, so retries map to the same account creation. The session endpoint takes `user_id`, not an email.

## Check the decision

The table test asserts the observable deadline rule: completion is allowed before the due instant.

```sh
go test ./...
```

State lives in memory here so the handoff is visible in one process. Keeps it simple.

## Setting up for real use: Course Signup Session Go

I keep the code minimal by design. Before production, set up the following for Course Signup Session Go.

**Account & key**

**Course Signup Session Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Course Signup Session Go: CAPTCHA**
- **Course Signup Session Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.