## create database http api
> To create a database send a POST request to the /query endpoint and set the URL parameter q to CREATE DATABASE <new_database_name>. The example below sends a request to InfluxDB running on localhost and creates the database mydb:
``` zsh
$ curl -i -XPOST http://host:8086/query --data-urlencode "q=CREATE DATABASE mydb"
```

## writing data http api
> The HTTP API is the primary means of writing data into InfluxDB, by sending POST requests to the /write endpoint. The example below writes a single point to the mydb database. The data consist of the measurement cpu_load_short, the tag keys host and region with the tag values server01 and us-west, the field key value with a field value of 0.64, and the timestamp 1434055562000000000. 
``` zsh
$ curl -i -XPOST 'http://localhost:8086/write?db=mydb' --data-binary 'cpu_load_short,host=server01,region=us-west value=0.64 1434055562000000000'
```