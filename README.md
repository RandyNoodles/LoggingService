# Overview
A simple TCP logging service:
- Uses goroutines to handle clients concurrently
- Message format is user-definable via JSON schema
- Configurable logs
  - column order
  - delimiters
  - JSON
- Configurable abuse prevention:
  - Messages/per client per min
  - Blacklist duration / permanent blacklist
  - Malformed request limit before blacklisting
---
# Logging Message Protocol
## Incoming Requests
Logs must be sent in `json` format.
The exact format of the `json` messages is defined on startup using the schema referenced in [`incoming_json_schema`](###protocol_settings)

Messages that do not validate successfully against the [`incoming_json_schema`](###protocol_settings) will be rejected, 
and a strike will be added against the client IP under.
Once a client sends `n` messages (defined in `bad_message_blacklist_threshold`), the IP address will be blacklisted.
## Server Response
Server will respond with a single message to all incoming requests.
**Format:**
`{"success": {type: boolean}, "message": {type: string, maxLength: 1024}`

`success` property will indicate if the log was successfully written to file
`message` will carry further details such as:
- The IP has been blacklisted
- The IP is blacklisted for `N` more seconds
- The IP has exceeded it's message rate limit
- Internal server error

# Server Internally-defined Fields
The server will internally determine the `timestamp` and `source_ip` of an incoming log for the purposes of abuse prevention.

These fields can be added to logfile output by including either `"timestamp"` or `"source_ip"` explicitly via the `coulmn_order` property of [`logfile_settings`](###logfile_settings)

**Note:** if the client wishes to define `timestamp` or `source_ip` client-side, they MUST use a different naming convention.

# Abuse Prevention
Some abuse prevention settings can be configured under `protocol_settings` found within `config.json`.
## Message Rate Limiting
- Messages received from a given IP are tracked on a per-minute basis
- The threshold can be defined in `config.json` under `messages_per_ip_per_minute`
- Each message sent that **exceeds** this threshold counts as a malformed request, potentially leading to the IP being blacklisted, as per the `bad_message_blacklist_threshold`

## Malformed Requests
- Malformed requests are not written to the logfile
- Each malformed request from an IP will increment that IP's `bad_message_blacklist_threshold` counter.
- Once the `bad_message_blacklist_threshold` has been reached, an IP is blacklisted.
## Blacklisted IPs
- No received logs will be written to file
- If `blacklist_permanent` is set to `false`:
	- IPs will receive a response with the remaining duration of their blacklisted status
	- IPs will be un-blacklisted upon the first message received past the configured `blacklist_duration_seconds`
- If `blacklist_permanent` is set to `true`:
	- IPs will be notified they are blacklisted
	- IPs will not be un-blacklisted until server reboot.
- User can pre-configure a list of `blacklisted_ip` values in `config.json`
# Config
All configuration must be done via `[root]/config.json`
For more explicit formatting, see `config_schema.json`

## Individual Settings
### server_settings
**IP**: The ip address for the listener
**Port**: The port for the listener

### logfile_settings
`path`: Path to the logfile where all logs will be written

`format`: "json", or "plaintext". If "plaintext" selected, will use the delimiters noted below.

`plaintext_field_delimiter`: A string of chars to be written to log between each field in a record

`plaintext_entry_delimiter`: A string of chars to be written between each record

`column_order`: The order of columns to be written to log. (SEE USAGE BELOW)

#### `column_order` Usage
- `column_order` determines what order the fields are written to logfile.
- Any field names not found in the `incoming_json_schema`, or in the `server default fields` will throw an error on startup.
	- Server default fields: `timestamp`, `source_ip`
- If a field name is omitted from this list, it will not be written to the logfile
### protocol_settings
`incoming_json_schema`: Relative file path of the JSON schema used to validate incoming JSON log messages. Note: `timestamp` and `source_ip` fields are server-generated.

`messages_per_ip_per_minute`: The number of messages an IP can send per minute before they are blacklisted.

`bad_message_blacklist_threshold`: The number of malformed logs sent before an IP is blacklisted.

`blacklisted_ips`: Array of user-defined IPs blacklisted upon startup. Format must be IPv4

`blacklist_permanent`: If `true`, blacklisted IPs will never be reset.

`blacklist_duration_seconds`: If `blacklist_permanent` is set to `false`, IPs will be removed from the blacklist on their first message attempt after N seconds.

### error_handling
`invalid_message`: On invalid message format, either `redirect_to_error_log`, which will log the formatting error for later review. Or `ignore`, meaning client will be notified, but error is not logged.

`error_log_path`: Path to file where errors are logged.
