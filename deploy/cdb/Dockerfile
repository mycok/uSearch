FROM cockroachdb/cockroach

COPY linkgraph/store/cdb/migrations /migrations
COPY deploy/bootstrap_db.sh .
ENTRYPOINT [ "bash", "./bootstrap_db.sh" ]
