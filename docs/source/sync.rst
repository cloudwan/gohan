=================
Sync with backend
=================

Gohan stores event log for create, update and delete db operation.
This will be done in the db transaction, so we can assume consistensy
on this event log data with resource data.

Event log data is something like this. (see schema in gohan.json)

- id
- type
- updated time
- path
- body

Gohan server will select one master node using etcd backend CAS API.
Then master node will poll event log table, then push to the backend.

We may support mysql binlog api for better performance in future.
