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
| Topic Prefix | Prepended to all structured topics | `shutterbase` |
| WLED Device Topic | Direct WLED control (see below) | `wled/device1` |

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

### Option A: Direct WLED Control (Recommended)

Shutterbase can publish **directly** to your WLED device's API topic — no bridge or automation tool needed.

#### 1. Enable MQTT on WLED

In WLED UI: **Config > Sync Interfaces > MQTT**:
- Enable MQTT
- Set the broker IP/hostname and port (default `1883`)
- Note the **MQTT device topic** (default: `wled/<device_id>`, e.g. `wled/a4cf12fa54b3`)

#### 2. Enter WLED Device Topic in Shutterbase

In Project Settings > MQTT:
- Set **WLED Device Topic** to your WLED's device topic (e.g. `wled/a4cf12fa54b3`)
- Enable events and set preset numbers

#### 3. Create Presets in WLED

In WLED UI: **Presets > +**:
- Create presets with the effects you want for each event
- Note the preset IDs (1, 2, 3, etc.)

#### How It Works

When an event fires, Shutterbase publishes two messages:
1. **Structured topic** (for HA, Node-RED, custom apps): `{prefix}/{projectId}/upload/{uploadId}/{event}`
2. **WLED topic** (direct control): `{wledDeviceTopic}/api` with payload `{"preset": N}`

WLED subscribes to `{wledDeviceTopic}/api` natively — it receives the command and triggers the preset immediately.

```
Upload approved → Shutterbase publishes {"preset":4} to wled/device1/api → WLED triggers preset 4
```

### Option B: Bridge / Automation Tool

If you want more control or have multiple WLED devices, use a bridge.

**Mosquitto Topic Bridge**

Add to `mosquitto.conf` to forward Shutterbase topics to WLED:
```
topic shutterbase/+/upload/+/+ in 0
```

Then use an automation tool to filter and forward to WLED's API topic.

**Home Assistant / Node-RED**

Subscribe to `shutterbase/#` and create automations:

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

### Simple: Flash on Approval (Direct WLED)

1. WLED: create preset 4 (solid gold)
2. Shutterbase Project Settings > MQTT:
   - Broker: `tcp://mosquitto:1883`
   - WLED Device Topic: `wled/a4cf12fa54b3`
   - Events: enable **Approved**, preset `4`
3. When a reviewer approves an upload, WLED flashes gold — no bridge needed

### Multi-Event: Full Review Flow

1. WLED presets:
   - Preset 1: Green pulse (upload created)
   - Preset 3: Blue pulse (ready for review)
   - Preset 4: Gold flash (approved)
   - Preset 5: Red flash (rejected)
2. Shutterbase: enable all four events with matching preset numbers
3. WLED reacts to each stage of the review process

### Tag-Based Effects

1. Enable **Tag assigned**, trigger tags: `winner, highlight`
2. Preset 7: Rainbow effect
3. When a `winner` tag is assigned, WLED shows rainbow

### Multi-Device (with Bridge)

Different WLED devices react to different events:

- **Office WLED** (`wled/office`): preset 4 on approvals — gold flash
- **Studio WLED** (`wled/studio`): preset 3 on ready — blue pulse
- **Party WLED** (`wled/party`): preset 7 on tag assigned — rainbow

Use the **Topic Prefix** for structured topics + Home Assistant/Node-RED to route to each device.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| No messages received | Is the MQTT broker running? Is the Broker URL correct in project settings? |
| WLED not reacting (direct) | Is the WLED Device Topic correct? Is the preset number set? Is WLED connected to the same broker? |
| WLED not reacting (bridge) | Is the bridge configured? Is Home Assistant/Node-RED subscribed to the right topic? |
| Connection lost warnings | Check broker logs, network connectivity, credentials |
| Events not firing | Is the event toggle enabled in project settings? |
| Tag triggers not working | Is the tag name in the trigger list exactly (case-sensitive)? |
| WLED preset not found | Does the preset ID exist in WLED? Check WLED Presets page. |
