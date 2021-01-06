# Context Pointer to Transaction (cptx)

cptx is a wrapper of `sqlx` adjusted to Clean Architecture. it splits the transaction from main db operation as the same level of object to achive the principle of `no tech on business/usecase layer`. 

## How to use

1. [the storage level](https://github.com/iDevoid/stygis/tree/main/internal/storage/persistence)
2. [the business/usecase level](https://github.com/iDevoid/stygis/tree/main/internal/module/user)
3. [the intiator/cmd level](https://github.com/iDevoid/stygis/tree/main/initiator)

### source : https://github.com/iDevoid/stygis

# Background

Separation business logic or usecase layer from any technological access is hard enough that we usually don't care about it again.
and yet, it is so hard to separate the transaction from business logic layer, but it is inappropriate to allow sql/database access to business logic above the storage layer.
But, isn't the Transaction object a techology? yea, no.. here is the analogy:

## analogy

`When user register, create a user account and default profile page, if failed then don't save any data.`

this flow is the business flow, and it uses transaction. that's why business logic holds the rights whether to proceed or cancel the data saving process. That means business/usecase layer should have access to transaction, BUT not to the database directly.

# why use context?
1. I hate to say this, but writing context as the first param `func Name(ctx context.Context)` becomes a habit, even tho the context is not being used
2. I don't want to change the usual thing, meaning adding weird type param to my function is a no no.
3. I want to hold the principle of `no tech on business/usecase layer`

## did you know?
if you have 5 layers of architecture 

## size in bytes
since all of the operation fully runs on pointer, you don't have to worry about the memory usage.
Inside the context itself is the pointer of ptx (wrapped *sqlx.Tx) being store. 
Meaning it is the address you bring, not the transaction inside the context.

```
Begin size of ctx before hold tx: *fasthttp.RequestCtx, 16
Begin size of ctx after hold tx: *context.valueCtx, 16
Begin size of ptx: postgres.ptx, 8
Begin size of ptx: *sqlx.Tx, 8
query row size of ctx: *context.valueCtx, 16
```

###  context is useful