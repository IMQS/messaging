# Messaging

The messaging service listens for incoming requests to send SMS messages and may possibly also include email 
messages in future. Integrating with various SMS providers are supported, and all message 
activity is recorded in a SQL database.   


## Features

The messaging service supports the following:

- web server to listen for HTTP `GET` or `POST` requests on a configurable port
- authentication of user roles via the `serviceauth` package
- processing of lists of mobile numbers (MSISDNs), cleaning and/or discarding invalid numbers and duplicates.
- configurable SMS provider integration - a MockProvider is included for testing
- splitting of large send requests into smaller batches, as required by some SMS providers
- send message and clean mobile numbers to SMS provider through API
- logging of all messages and send logs in SQL tables
- configurable optional polling to retrieve the delivery status for messages
 
## API calls

### **sendSMS**
Sends a message to the mobile numbers included in the JSON POST request.

* **URL**

  /sendsms

* **Method:**

  `POST`
  
* **Data Params**

```json
{ "message": "text message to send",
  "msisdns": [
  		"0830000000",
  		"27840000000"
  	     ]
}
```
* **Success Response:**

  * **Code:** 200 <br />
    **Content:** 
```json
{ "refNumber": "412",
  "validNumbers": 5,
  "invalidNumbers": 2,
  "sendSuccess": true,
  "statusDescription": "",
  "messagesSent": 5 }
```
 
* **Error Response:**

  * **Code:** 200 <br />
    **Content:** 
```json
{ "refNumber": "412",
  "validNumbers": 5,
  "invalidNumbers": 2,
  "sendSuccess": false,
  "statusDescription": "Error 301: Out of credit",
  "messagesSent": 0 }
```

### **messageStatus**
Retrieves the delivery status of the last delivered message for a specific mobile number.

* **URL**

  /messagestatus/:mobileNumber

* **Method:**

  `GET`
  
* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `delivered` | `failed` | `sent`
 
* **Error Response:**

  * **Code:** 200 <br />
    **Content:** `Error: no data available`

* **Status Descriptions:**

	* **delivered:** Message has successfully been delivered to the mobile phone <br />
	* **failed:** Message could not be delivered to the mobile phone <br />
	* **sent:** Message has been sent but has not reached the handset yet.  It could still fail or be delivered.


### **Normalize**
Processes a list of mobile numbers, cleaning, formatting and removing duplicates.

* **URL**

  /normalize

* **Method:**

  `POST`
  
* **Data Params**

   **Required:**

```json
{ "msisdns": [
  		"0830000000",
  		"27840000000",
  		"numbersToClean",
  		"0820000000"
  	     ]
}
```

* **Success Response:**

  * **Code:** 200 <br />
    **Content:** 
```json
[ 
  "27830000000",
  "27840000000",
  "27820000000"
]
```
 
* **Error Response:**

  * **Code:** 401 UNAUTHORIZED <br />

----------

## Configuration

```
{
	"HTTPPort": 2016,  			    // Port to bind to for the HTTP server
	"Logfile": "c:\\imqsvar\\logs\\services\\messaging\\messaging.log",
	"smsProvider": {
		"name": "MockProvider",		// Name of provider.  Will be used to determine function to call
		"enabled": true,			// Enable or disable sending of SMS for testing
		"token": "12345",			// Auth token to use for sending
		"maxMessageSegments": 1,	// Max message segments to send. Each segment is 160 characters
		"maxBatchSize": 500,  		// Max number of messages to send per batch 
		"countries": ["ZA", "BW"]	// Allow sending to countries listed. Incompatible numbers will be discarded 
	},
	"authentication": {
		"service": "serviceauth",	// Authentication system to use. Implement new service in auth.go 
		"enabled": true			// Enable or disable user authentication
	},
	"deliveryStatus": {
		"enabled": true,			// Enable or disable delivery status retrieval
		"updateInverval": "15m"		// Amount of minutes between retrieval of delivery status  
	},
	"dbConnection": {
		"Driver": "postgres",		// Only Postgres implemented at this stage
		"Host": "localhost",		// DB hostname
		"Port": 5432,				// DB port
		"Database": "messaging",	// Database name to use.  Will be created if does not exist
		"User": "",					// DB user. Ensure user has permission to create databases 
		"Password": "",				// DB user password
		"SSL": false				// Enable or disable SSL for DB access
	}
}

```
