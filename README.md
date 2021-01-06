# Context Pointer to Transaction (cptx)

cptx is a wrapper of `sqlx` adjusted to Clean Architecture. it splits the transaction from main db operation as the same level of object to achive the principle of `no tech on business/usecase layer`. 

## How to use

1. [the storage layer](https://github.com/iDevoid/stygis/tree/main/internal/storage/persistence)
2. [the business/usecase layer](https://github.com/iDevoid/stygis/tree/main/internal/module/user)
3. [the intiator/cmd layer](https://github.com/iDevoid/stygis/tree/main/initiator)

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
if you have 5 layers of architecture, and all accessed function from high to low level layer, you have at least 5 pointers of the same context. because the behavior of function parameter is copying the param, you have to have star (*) symbol before writing the type of param to give the pointer instead of copying to new pointer.
Golang Garbage Collector is powerful enough to handle such a small things already. But deeply I am thinking about the habit of writing context as param for nothing is kinda sad. Context is useful, that's why I use the context as the "pipe" to bring the pointer of transaction to lower level layer without bring the technology to higher level layer.

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