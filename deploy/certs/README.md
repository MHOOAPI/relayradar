# Put fullchain.pem and privkey.pem here.

For a throwaway self-signed certificate:

```bash
openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
  -keyout privkey.pem -out fullchain.pem \
  -subj "/CN=relay.example.com"
```

For production, point these files at your Let's Encrypt (or other CA) certificate,
for example by copying from `/etc/letsencrypt/live/<domain>/`.
