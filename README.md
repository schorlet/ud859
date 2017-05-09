ud859 [![GoDoc](https://godoc.org/github.com/schorlet/ud859?status.png)](https://godoc.org/github.com/schorlet/ud859)
===

This project is an implementation of the udacity course at
[http://udacity.com/course/ud859](http://udacity.com/course/ud859) with the Go programming language.

~~The application is running at [https://ud859-go.appspot.com](https://ud859-go.appspot.com) and exposes a REST API to manage conferences using the Cloud Endpoints feature of Google App Engine.~~


## Feedback

+ **For the curious**

~~The conference API endpoint is also queryable from [apis-explorer.appspot.com](https://apis-explorer.appspot.com/apis-explorer/?base=https://ud859-go.appspot.com/_ah/api).~~

+ **When developping the API endpoint**

Make sure that your endpoint is readable by the [discovery service](https://developers.google.com/discovery/). This URL [http://localhost:8080/_ah/api/discovery/v1/apis/conference/v1/rest](http://localhost:8080/_ah/api/discovery/v1/apis/conference/v1/rest) should return the service endpoint description.

+ **Testing**

You can run the tests with ```go test``` (no need for ```goapp test```) and take advantage of parallel subtests of go1.7.

+ **My Eureka moment**

When I managed to fake the endpoint authentication in the tests.


