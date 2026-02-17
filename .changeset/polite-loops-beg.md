---
"chainlink-deployments-framework": minor
---

feat(chain): update Canton support to support clients and authentication

- Add gRPC service clients to Canton chain output
- Add UserID and PartyID to Canton chain output
- Remove JWTProvider and replace with oauth2.TokenSource for authentication
