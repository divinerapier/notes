# Querying Data with the HTTP API
## Querying data using the HTTP API 
The HTTP API is the primary means for querying data in InfluxDB (see the <a href="https://docs.influxdata.com/influxdb/v1.1/tools/shell/">command line interface</a> and <a href"https://docs.influxdata.com/influxdb/v1.1/tools/api_client_libraries/">client libraries</a> for alternative ways to query the database). 

To perform a query send a `GET` request to the `/query` endpoint, set the URL parameter `db` as the target database, and set the URL parameter `q` as your query. The example below uses the HTTP API to query the same database that you encountered in Writing Data. 
``` zsh
 curl -G 'http://localhost:8086/query?pretty=true' --data-urlencode "db=mydb" --data-urlencode "q=SELECT \"value\" FROM \"cpu_load_short\" WHERE \"region\"='us-west'"
 ```
InfluxDB returns JSON. The results of your query appear in the `"results"` array. If an error occurs, InfluxDB sets an `"error"` key with an explanation of the error. 
``` json
 {
    "results": [
        {
            "series": [
                {
                    "name": "cpu_load_short",
                    "columns": [
                        "time",
                        "value"
                    ],
                    "values": [
                        [
                            "2015-01-29T21:55:43.702900257Z",
                            0.55
                        ],
                        [
                            "2015-01-29T21:55:43.702900257Z",
                            23422
                        ],
                        [
                            "2015-06-11T20:46:02Z",
                            0.64
                        ]
                    ]
                }
            ]
        }
    ]
}
```
>Note: Appending `pretty=true` to the URL enables pretty-printed JSON output. While this is useful for debugging or when querying directly with tools like `curl`, it is not recommended for production use as it consumes unnecessary network bandwidth.
## Multiple queries
Send multiple queries to InfluxDB in a single API call. Simply delimit each query using a semicolon, for example:
``` zsh
 curl -G 'http://localhost:8086/query?pretty=true' --data-urlencode "db=mydb" --data-urlencode "q=SELECT \"value\" FROM \"cpu_load_short\" WHERE \"region\"='us-west';SELECT count(\"value\") FROM \"cpu_load_short\" WHERE \"region\"='us-west'"
```
returns:
``` json
 {
    "results": [
        {
            "series": [
                {
                    "name": "cpu_load_short",
                    "columns": [
                        "time",
                        "value"
                    ],
                    "values": [
                        [
                            "2015-01-29T21:55:43.702900257Z",
                            0.55
                        ],
                        [
                            "2015-01-29T21:55:43.702900257Z",
                            23422
                        ],
                        [
                            "2015-06-11T20:46:02Z",
                            0.64
                        ]
                    ]
                }
            ]
        },
        {
            "series": [
                {
                    "name": "cpu_load_short",
                    "columns": [
                        "time",
                        "count"
                    ],
                    "values": [
                        [
                            "1970-01-01T00:00:00Z",
                            3
                        ]
                    ]
                }
            ]
        }
    ]
}
```
## Other options when querying data
### Timestamp Format 
Everything in InfluxDB is stored and reported in UTC. By default, timestamps are returned in RFC3339 UTC and have nanosecond precision, for example `2015-08-04T19:05:14.318570484Z`. If you want timestamps in Unix epoch format include in your request the query string parameter `epoch` where `epoch=[h,m,s,ms,u,ns]`. For example, get epoch in seconds with:
``` zsh
 curl -G 'http://localhost:8086/query' --data-urlencode "db=mydb" --data-urlencode "epoch=s" --data-urlencode "q=SELECT \"value\" FROM \"cpu_load_short\" WHERE \"region\"='us-west'"
 ```

### Authentication 
Authentication in InfluxDB is disabled by default. See Authentication and Authorization for how to enable and set up authentication. 
### Maximum Row Limit 
InfluxDB will limit the maximum number of returned results to prevent itself from running out of memory while it aggregates the results. This is set to 10,000 by default and can be configured by modifying `max-row-limit` in the `http` section of the configuration file.     

The maximum row limit only applies to non-chunked queries. Chunked queries can return an unlimited number of points. 
### Chunking 
Chunking can be used to return results in streamed batches rather than as a single response by setting the query string parameter `chunked=true`. Responses will be chunked by series or by every 10,000 points, whichever occurs first. To change the maximum chunk size to a different value, set the query string parameter `chunk_size` to a different value. For example, get your results in batches of 20,000 points with:
``` zsh
 curl -G 'http://localhost:8086/query' --data-urlencode "db=deluge" --data-urlencode "chunked=true" --data-urlencode "chunk_size=20000" --data-urlencode "q=SELECT * FROM liters"
 ```
### InfluxQL
Now that you know how to query data, check out the <a href="https://docs.influxdata.com/influxdb/v1.1/query_language/data_exploration/">Data Exploration page</a> to get acquainted with InfluxQL.