# xtoken
Package xtoken is a globally unique token generator library. 
Inspired by [xid](https://github.com/rs/xid), xtoken prioritizes randomness and security over sortability, making it 
ideal for session tokens, API keys. Unlike xid's fixed structure, xtoken introduces a novel offset-based encoding, 
where a few fixed bytes store a random offset value, shuffling the token's internal components (timestamp and counter) to 
prevent predictability. 


- **Size**: 12 bytes (raw), 32 chars (encoded).
- **Randomized**: Non-predictable with random offset encoding.
- **Components**:
  - 4-byte value representing the seconds since the Unix epoch,
  - 3-byte machine identifier,
  - 2-byte process id, and
  - 3-byte counter, starting with a random value.
## Install
```shell
go get github.com/zdz1715/xtoken
```
## Usage
```go
gtoken := xtoken.New()

println(gtoken.String())
// Output: VKEoZ3FCqGChUJNBWAaq1WDrXLIpIaPY
```
### Get embedded info:xtoken
```go
gtoken.Machine()
gtoken.Pid()
gtoken.Time()
gtoken.Counter()
```
### Expire:
To quickly check if a token has expired, you can set its timestamp to an expiration time:

```go
// Generate a token that expires in 7 days
token := xtoken.NewWithTime(time.Now().Add(7 * 24 * time.Hour))
println(token.String()) // e.g., "VKEoZ3FCqGChUJNBWAaq1WDrXLIpIaPY"

// Check if expired
t, err := xtoken.FromString(token.String())
if err != nil {
  println("Token invalid")
  return err
}
if time.Now().After(t.Time()) {
  println("Token expired")
} else {
  println("Token valid")
}
```

## Comparison with xid:
- [xid](https://github.com/rs/xid): Time-ordered, sortable IDs with predictable structure (20-char base32).
- xtoken: Random, non-sortable tokens with offset-based encoding (32-char, increased randomness).

## Benchmark
```shell
BenchmarkNew-10    	14123192	        87.55 ns/op	       0 B/op	       0 allocs/op
BenchmarkNew-10    	14123192	        87.55 ns/op	       0 B/op	       0 allocs/op
```




