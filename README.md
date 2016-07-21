# Messaging

The messaging service listens for incoming requests to send SMS messages and may possibly also include email 
messages in future. Integrating with various SMS providers are supported, and all message 
activity is recorded in a SQL database.   


## Features

The messaging service supports the following:

- web server to listen for HTTP `GET` or `POST` requests on a configurable port
- authentication of user roles via the `serviceauth` package
- processing of lists of mobile numbers (MSISDNs), cleaning and/or discarding invalid numbers and duplicates.
- configurable SMS provider integration 
- splitting of large send requests into smaller batches, as required by some SMS providers
- send message and clean mobile numbers to SMS provider through API
- logging of all messages and send logs in SQL tables
- configurable optional polling to retrieve the delivery status for messages 
 
## API calls

### **sendSMS**
Sends a message to the mobile numbers included in the form-data of the POST request.

* **URL**

  /sendSMS

* **Method:**

  `POST`
  
* **Data Params**

   `message=[string]` -> text message to send

   `msisdns=[string, csv]` -> list of mobile numbers in CSV format

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

  /messageStatus/:mobileNumber

* **Method:**

  `GET`
  
* **Success Response:**

  * **Code:** 200 <br />
    **Content:** `004`
 
* **Error Response:**

  * **Code:** 200 <br />
    **Content:** `Error: no data available`

### **Normalize**
Processes a list of mobile numbers, cleaning, formatting and removing duplicates.

* **URL**

  /normalize

* **Method:**

  `POST`
  
* **Data Params**

   **Required:**

   `msisdns=[string, csv]` -> list of mobile numbers in CSV format

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
	"smsProvider": {
		"name": "MockProvider",		// Name of provider.  Will be used to determine function to call
		"enabled": true,			    // Enable or disable sending of SMS for testing
		"token": "12345",			// Auth token to use for sending
		"maxBatchSize": 500,  		// Max number of messages to send per batch 
		"custom1": "custom data",	// Custom field for SMS provider API implementation
		"custom2": "custom data",	// Custom field for SMS provider API implementation
		"custom3": "custom data"	// Custom field for SMS provider API implementation
	},
	"authentication": {
		"service": "serviceauth",	// Authentication system to use. Implement new service in auth.go 
		"enabled": false			// Enable or disable user authentication
	},
	"deliveryStatus": {
		"enabled": true,			// Enable or disable delivery status retrieval
		"updateInverval": 15		// Amount of minutes between retrieval of delivery status  
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
