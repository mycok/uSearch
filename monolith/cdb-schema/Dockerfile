FROM cockroachdb/cockroach
COPY linkgraph/store/cdb/migrations /migrations
COPY monolith/cdb-schema/bootstrap_db.sh .

ENTRYPOINT [ "bash", "./bootstrap_db.sh" ]
