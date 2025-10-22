---
"chainlink-deployments-framework": minor
---

feat(catalog): new datastore field in domain.yaml


Field `datastore` is introduced to configure in future where should the data be written to, either file(json) - current behaviour or remote on the catalog service. 
By default, this field will be set to file for backwards compatibility.
