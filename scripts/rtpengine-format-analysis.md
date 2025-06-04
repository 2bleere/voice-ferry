# RTPEngine Bencode Protocol Analysis

## What we've observed in RTPEngine logs:

### ❌ FAILING Formats (showing "no cookie" error):

1. **No cookie at all:**
   ```
   d7:command43:{"command":"ping","call-id":"","Data":null}e
   ```

2. **Cookie in JSON instead of bencode level:**
   ```
   d7:command73:{"command":"ping","call-id":"","cookie":"b2bua-health-check","Data":null}e
   ```

### ✅ Expected Format (based on RTPEngine documentation):

The bencode format should be:
```
d6:cookie<cookie_length>:<cookie_value>7:command<json_length>:<json_string>e
```

Example:
```
d6:cookie16:abc123def45678907:command25:{"command":"ping"}e
```

## Key Issues Identified:

1. **Data field**: The `"Data":null` field should not be in the JSON
2. **Cookie placement**: Cookie must be at bencode level, NOT in JSON
3. **Clean JSON**: Only include necessary fields in the JSON command

## Test Commands:

Run the Python test script:
```bash
python3 scripts/test-rtpengine-bencode.py
```

Run the Go test (requires compilation):
```bash
cd scripts && go run test-bencode-format.go
```

## Expected Working Format:

```go
cookie := generateCookie()
cmd := Command{
    Command: "ping", 
    CallID:  "",
}
cmdBytes, _ := json.Marshal(cmd)
bencode := fmt.Sprintf("d6:cookie%d:%s7:command%d:%se", len(cookie), cookie, len(cmdBytes), cmdBytes)
```

Should produce:
```
d6:cookie16:a1b2c3d4e5f6789a7:command25:{"command":"ping","call-id":""}e
```
