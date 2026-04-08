# evil-behavior-exercised

## Evidence

Five evil behaviors are implemented in `internal/player/behavior.go`:
1. Bad moves (ChaosRate=0.35): R26 confirms reached
2. Out-of-turn moves (OutOfTurnRate=0.20): R25 confirms reached
3. Malformed JSON (MalformedRate=0.55 of chaos): R11 confirms invalid payloads received
4. Extra connections (ExtraConnectRate=0.18): R27 confirms reached
5. Queue abandonment (QueueAbandonRate=0.16): R16 confirms reached

R12 confirms semantically invalid moves are received (the non-malformed chaos path).

## Instrumentation Status

**FULLY COVERED** — R16, R25, R26, R27, R11, R12 collectively cover all evil behaviors.
