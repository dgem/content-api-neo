# content-api-neo
Experimental content writer backed by neo4j

Example use:

- Install neo4j and start it up locally


- Install content-api-neo as follows:

```
go get -u github.com/Financial-Times/content-api-neo
```


- Run content-api-neo :
```
content-api-neo
```

- Insert an content :

```
curl  -d'{"uuid":"b967da4e-c28a-11e4-ad89-00144feab7de", ....}' -XPUT localhost:8080/content/b967da4e-c28a-11e4-ad89-00144feab7de
```

- Or insert many content at once (note that this does not replace collection currently):

```
curl  -d'{"uuid":"xxx"...}{"uuid":"yyy"...}' -XPUT localhost:8080/content/
```

- Point your browser at http://localhost:7474 and explore your content.
