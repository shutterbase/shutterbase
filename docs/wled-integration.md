# WLED / MQTT Integration

Shutterbase can publish events to an MQTT broker when uploads move through the review flow. [WLED](https://kno.wled.ge/) devices (or any MQTT-aware smart-home system) can subscribe to these topics and react with light effects.

## How It Works

```
Photographer uploads photos
        │
        ▼
Shutterbase publishes MQTT message
        │
        ▼
MQTT Broker (e.g. Mosquitto)
        │
        ▼
WLED device receives message
        │
        ▼
Triggers preset (light effect)
```

1. A project admin configures the MQTT broker and selects which events trigger messages
2. When an event occurs (upload created, approved, rejected, etc.), Shutterbase publishes a JSON payload to a structured topic
3. WLED subscribes to the topic and calls the JSON API with the preset number from the payload

## Prerequisites

- **MQTT Broker**: [Mosquitto](https://mosquitto.org/) or any MQTT v3.1.1 / v5 broker
- **WLED Device**: running WLED with MQTT enabled (`Security & Setup > MQTT`)
- **Network**: Shutterbase server must be able to reach the broker

## Configuration

Open your project **Settings > General** tab. Scroll to **MQTT / WLED Integration** (project admin only).

### Broker Connection

| Field | Description | Example |
|-------|-------------|---------|
| Broker URL | MQTT broker address | `tcp://localhost:1883` |
| Client ID | Unique identifier for the connection | `shutterbase-myproject` |
| Username | Broker auth (leave empty if none) | `mqttuser` |
| Password | Broker auth (leave empty if none) | `secret` |
| Topic Prefix | Prepended to all topics | `shutterbase` |

### Events

Toggle which events publish MQTT messages. Each event can have a WLED preset number — when the event fires, the payload includes `"preset": N` so WLED can trigger the corresponding effect.

| Event | When it fires | Topic |
|-------|---------------|-------|
| **Upload created** | New upload created by photographer | `{prefix}/{projectId}/upload/{uploadId}/created` |
| **Image uploaded** | Individual photo uploaded to an upload | `{prefix}/{projectId}/upload/{uploadId}/image-uploaded` |
| **Ready for review** | Photographer submits upload (`open` → `ready`) | `{prefix}/{projectId}/upload/{uploadId}/ready` |
| **Approved** | Reviewer accepts upload (`ready` → `reviewed`) | `{prefix}/{projectId}/upload/{uploadId}/approved` |
| **Rejected** | Reviewer sends upload back (`ready/reviewed` → `open`) | `{prefix}/{projectId}/upload/{uploadId}/rejected` |
| **Image rejected** | `rejected` tag assigned to an image | `{prefix}/{projectId}/upload/{uploadId}/image-rejected` |
| **Tag assigned** | Any tag in the trigger list is assigned | `{prefix}/{projectId}/upload/{uploadId}/tag-assigned` |

### Tag Triggers

When **Tag assigned** is enabled, specify which tag names trigger an MQTT message. Comma-separated, e.g.:

```
error, vip, highlight, winner
```

Only exact matches fire — assigning `error` triggers, assigning `error-fixed` does not.

## Payload Reference

All payloads are JSON. Common fields:

```json
{
  "preset": 3
}
```

### Event-Specific Payloads

**upload.created**
```json
{
  "uploadName": "Morning Session",
  "userId": "abc123",
  "preset": 1
}
```

**image.uploaded**
```json
{
  "imageId": "img_abc123",
  "fileName": "DSC_0042.ARW",
  "uploadId": "up_xyz789",
  "userId": "abc123",
  "preset": 2
}
```

**ready**
```json
{
  "uploadName": "Morning Session",
  "oldState": "open",
  "newState": "ready",
  "userId": "abc123",
  "preset": 3
}
```

**approved**
```json
{
  "uploadName": "Morning Session",
  "oldState": "ready",
  "newState": "reviewed",
  "userId": "reviewer1",
  "preset": 4
}
```

**rejected**
```json
{
  "uploadName": "Morning Session",
  "oldState": "ready",
  "newState": "open",
  "userId": "reviewer1",
  "preset": 5
}
```

**image-rejected**
```json
{
  "imageId": "img_abc123",
  "fileName": "DSC_0042.ARW",
  "rejectedBy": "reviewer1",
  "preset": 6
}
```

**tag-assigned**
```json
{
  "imageId": "img_abc123",
  "fileName": "DSC_0042.ARW",
  "tagName": "vip",
  "userId": "tagger1",
  "preset": 7
}
```

## WLED Setup

### 1. Enable MQTT on WLED

In WLED UI: **Security & Setup > MQTT**:
- Enable MQTT
- Set the broker IP/hostname and port (default `1883`)
- Set MQTT topic (e.g. `wled/device1`) — this is WLED's own topic, not the Shutterbase topic

### 2. Subscribe to Shutterbase Topics

WLED automatically subscribes to `wled/+` and its own topic. To receive Shutterbase messages, you need a **bridge** — either:

**Option A: MQTT bridge in Mosquitto** (recommended)

Add to `mosquitto.conf`:
```
topic shutterbase/+/upload/+/+ in 0
```

Then in WLED, use a relay or use an MQTT automation tool (Node-RED, Home Assistant) to forward messages.

**Option B: Use WLED's MQTT update topic**

Configure WLED to listen on a wildcard. In WLED's MQTT settings, set the **MQTT device topic** to match your Shutterbase prefix, or use an automation tool to bridge.

**Option C: Home Assistant / Node-RED**

The most flexible approach — subscribe to `shutterbase/#` and create automations:

```yaml
# Home Assistant example
automation:
  - alias: "WLED on upload approved"
    trigger:
      platform: mqtt
      topic: "shutterbase/myproject/upload/+/approved"
    action:
      service: mqtt.publish
      data:
        topic: "wled/device1/api"
        payload: '{"preset":4}'
```

### 3. WLED Presets

Create presets in WLED (WLED UI > Presets > +) for each event you want to visualize:

| Preset # | Suggested Effect | Use Case |
|----------|------------------|----------|
| 1 | Solid green | Upload created |
| 2 | Rainbow chase | Photo uploaded |
| 3 | Pulse blue | Ready for review |
| 4 | Solid gold | Approved |
| 5 | Red flash | Rejected |
| 6 | Red pulse | Image rejected |
| 7 | Rainbow | Tag assigned |

## Examples

### Simple: Flash on Approval

1. Project Settings > MQTT:
   - Broker: `tcp://mosquitto:1883`
   - Topic prefix: `shutterbase`
   - Events: enable **Approved**, preset `4`
2. WLED preset 4: Solid gold, 30s fade-out
3. When a reviewer approves an upload, WLED flashes gold

### Advanced: Tag-Based Effects

1. Enable **Tag assigned**, trigger tags: `winner, highlight`
2. Preset `7`: Rainbow effect
3. When a `winner` tag is assigned, WLED shows rainbow

### Multi-Device

Different WLED devices subscribe to different topic patterns:

- **Office WLED**: `shutterbase/+/upload/+/approved` — gold flash on approvals
- **Studio WLED**: `shutterbase/+/upload/+/ready` — blue pulse when work arrives
- **Party WLED**: `shutterbase/+/upload/+/tag-assigned` — rainbow on any tag

## Troubleshooting

| Symptom | Check |
|---------|-------|
| No messages received | Is `MQTT_BROKER` set in server env? Is the broker running? |
| Messages but no WLED reaction | Is WLED subscribed to the right topic? Is the preset number correct? |
| Connection lost warnings | Check broker logs, network connectivity, credentials |
| Events not firing | Is the event toggle enabled in project settings? |
| Tag triggers not working | Is the tag name in the trigger list exactly (case-sensitive)? |
