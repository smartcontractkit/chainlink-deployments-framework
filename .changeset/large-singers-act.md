---
"chainlink-deployments-framework": patch
---

fix(jd): keep wsrpc field as storage
    
Looks like WSRPC field cant be removed completely for now as Chainlink repo uses WSRPC field of the JDConfig as temporary storage for lookup later, it requires a refactor on the Chainlink side to address this, in the mean time to unblock the removal of wsrpc in the CLD, we temporary restore the storage functionality of the field.
