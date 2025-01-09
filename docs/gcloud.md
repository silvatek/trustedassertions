Google Cloud
============
The Trusted Assertions application is designed to be deployed as an OCI container to any container runtime platform. The initial platform is Google Cloud Run.

Services In Use
===============
* Google Cloud Run
* Google Cloud Build
* Firestore
* Logging
* Applicationb Load Balancer
* Cloud Storage
* Certificate Manager

The load balancer routes all unknown paths to the Cloud Storage bucket so that those requests don't wake up the Cloud Run instance if it is inactive.

The Cloud Storage bucket has an error page defined in its Website Configuration, for 404 response codes.

Abandoned Investigations
========================
I have tried a number of alternative approaches the current platform configuration that
didn't work out.

Cross-Project Application Load Balancer
---------------------------------------
Tried a cross-project ALB, but it won't accept the cross-project backend reference.

Google Cloud CDN
----------------
Couldn't get a non-zero cache hit rate.

Web Application Firewall
------------------------
Couldn't get it to reject malicious traffic.

Disable Scale-to-Zero
---------------------
All the requests from bots are keeping an instance alive most of the time, so I tried setting the minimum instances to 1 instead of zero. This seemed to change the billable instances to a minimum of 1 second per second though.

Tracing
-------
Couldn't figure out how to get spans created from logs.





