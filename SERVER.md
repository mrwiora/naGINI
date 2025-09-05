- write script to tempfile
- calculate sha256 sum
- convert sha256 to base32 encoded string
- create QR code based on that secret (base32 encoded sha256 sum)

- rename tempfile to last 8 characters of sha256 sum
- set password to first 8 characters
