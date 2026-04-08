# all-game-outcomes-observed

## Evidence

Three `Sometimes` assertions cover game outcomes:
- S11: `games sometimes end in draws`
- S12: `games sometimes end due to cancellation`
- S13: `games sometimes end with a winner` (P1Win or P2Win, combined)

The evil player workload increases the likelihood of cancellations (turn timeouts from invalid moves). Normal player workload produces wins and draws. Battleship games are less likely to draw (only if both players somehow can't complete).

## Gap

S13 combines P1Win and P2Win into one assertion. There's no separate assertion for P1Win vs P2Win. Since player assignment (P1 vs P2) is determined by connection order, and evil players may timeout more often as P1 or P2, the distribution may be skewed.

## Instrumentation Status

**MOSTLY COVERED** — S11, S12, S13 cover the major categories. Adding separate Sometimes for P1Win and P2Win would provide finer-grained coverage.
