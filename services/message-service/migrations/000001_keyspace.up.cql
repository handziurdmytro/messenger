CREATE KEYSPACE messenger
    WITH replication = {
        'class': 'NetworkTopologyStrategy',
        'datacenter1': 1
        }
     AND durable_writes = true;