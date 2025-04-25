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

### Comparison with xid:
- [xid](https://github.com/rs/xid): Time-ordered, sortable IDs with predictable structure (20-char base32).
- xtoken: Random, non-sortable tokens with offset-based encoding (32-char, increased randomness).



