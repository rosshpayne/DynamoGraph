# aws dynamodb batch-write-item \
#   --request-items file://data7a.json \
#     --return-consumed-capacity TOTAL
# aws dynamodb batch-write-item \
#   --request-items file://data7b.json \
#     --return-consumed-capacity TOTAL
# aws dynamodb batch-write-item \
#     --request-items file://data7c.json \
#     --return-consumed-capacity TOTAL
aws dynamodb batch-write-item \
  --request-items file://Types.1.json \
  --return-consumed-capacity TOTAL
# aws dynamodb batch-write-item \
#   --request-items file://Types.json \
#   --return-consumed-capacity TOTAL