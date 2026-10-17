CREATE KEYSPACE IF NOT EXISTS messenger
    WITH replication = {
        'class': 'NetworkTopologyStrategy',
        'datacenter1': 1
        }
     AND durable_writes = true
    AND tablets = {'enabled': false};
