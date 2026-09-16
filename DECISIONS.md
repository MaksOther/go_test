a. Lost create response
Before calling `POST /databases` the controller writes status.createRequestedAt with a plain `Update`. A stale `resourceVersion` makes that write fail, so two reconciles can't both send a create

After the call:

* 1)`201`: save the ID to status, keep the marker as history
* 2)`503` or connection refused: nothing was processed, so clear the marker and retry
* 3) Anything else (`500`, timeout, broken body): unknown result. State goes to `Unknown` and the controller never sends a second create for this resource

If the controller dies between sending the request and saving the ID, the next reconcile finds the marker with no ID and lands in `Unknown` too. Recovery is manual find the database in a support tool by its name (`<resource name>-<first 8 chars of UID>`) and set the `demo.example.com/database-id` annotation. The controller verifies it with `GET` and adopts it.



Costs -  roughly 15% of creates lose the response and need a human. Deleting the resource instead of adopting leaks the database, with only a log line to show for it. A crash just before the request goes out produces a false `Unknown`. And all of this assumes `503` really means "not processed"

b. What I'd change in the API
Add an idempotency key on create. The client sends a unique key (the resource UID), and a repeated `POST` with the same key returns the existing database instead of making a new one. That removes the `Unknown` state and the manual adoption entirely; create just gets retried. Lookup by name would work too, but it's a weaker version of the same thing.

The pitch to the owning team: right now every client of this API leaks paid databases when a response is lost, and support cleans them up by hand. Their side of the change is small (store key > ID, check before insert) and it's backwards compatible since the key is optional. I d bring the leaked - db numbers and offer to write the patch myself if they're short on time

c. Deletion keeps failing
Deletion blocks. The finalizer comes off only on `204` or `404`. Until then the state is `Deleting` with the last error in `message`, retried on the controller-runtime backoff

Giving up means a paid database nobody knows about, and with no list endpoint it can't be found again. A stuck resource is visible and costs nothing. If someone decides the database doesn't matter, they remove the finalizer by hand, so a person makes that call rather than a timeout.

One exception: a resource with no database ID (`Pending`, `Unknown`) has nothing to delete, so it's released immediately and the external name is logged



d. Editing `sizeGB`

`engine` and `sizeGB` are immutable via a CEL rule in the CRD (`self == oldSelf`). The API server rejects the edit with "sizeGB cannot be changed, the provisioning API has no resize operation", so the user sees it at `kubectl apply` and the spec never drifts from reality. A different size means a new resource. Downside: so does a typo

e) Left out, and what's next

Not done: `status.conditions` and events, RBAC and deployment manifests, leader election, periodic re-check of `Ready` databases (a database deleted outside the controller goes unnoticed until the next resync), automatic replacement of `FAILED` databases, and integration tests in the repo. I ran the flows through envtest with high failure rates locally, but only the unit tests are committed

Next: a `Ready` condition plus events, an envtest suite covering create, crash and delete, and deployment manifests with leader election

:)
