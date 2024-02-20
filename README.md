# Flood

Liquidity Provider bot for CL pools.

Flood creates and manages Concentrated Liquidity (CL) pool positions ensuring that capital deployed is done according to a useful set of rules.

The primary objective is to try to maintain two positions of quote and base assets that allows the owner of the bot to define the price(s) at which the quote and base are to be sold.

## Strategy Description

The strategy looks at the current theoretical price of sqASSET and the market price of sqASSET and adjust two LP positions accordingly.

### 1. Delta Between Mark and Theoretical Price

The strategy will first check the delta between the market and theoretical price of sqASSET. If the delta is greater than a specific threshold then it adjusts the positions.

However, if the delta is lower than the threshold, and the strategy is already LPing, it does nothing. If the strategy is not LPing already then it creates position nonetheless.

```
0 --------------- m ----- t --------------------- infty
```

### 2.
