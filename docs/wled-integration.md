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
WLED device receives message on {deviceTopic}/api
        │
        ▼
Triggers effect, preset, or raw JSON command
        │
        ▼
(optional) Auto-off after N seconds
```

1. A project admin configures the MQTT broker and selects which events trigger messages
2. When an event occurs (upload created, approved, rejected, etc.), Shutterbase publishes a command to the WLED device's API topic
3. Each event can trigger a **preset**, an **effect** (from a dropdown of 117+ built-in WLED effects), or **raw JSON** sent directly to WLED
4. An optional **auto-off duration** turns the strip off after N seconds

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

Toggle which events publish MQTT messages. Each event has three configuration options:

#### WLED Command Mode

For each enabled event, choose one of three modes:

| Mode | What it sends | Best for |
|------|---------------|----------|
| **Preset** | `{"preset": N}` | Users who have created presets in WLED |
| **Effect** | `{"seg": [{"fx": N}]}` | Quick setup — pick from 117+ built-in effects |
| **Raw JSON** | Your custom JSON | Full control — colors, brightness, segments, transitions |

**Priority**: Raw JSON > Effect > Preset. Only one mode is active per event.

#### Auto-Off Duration

Set a number of seconds (0 = disabled). After the effect triggers, Shutterbase waits that many seconds then sends `{"on": false}` to turn the strip off. This is useful for short visual cues (e.g. a 3-second flash when a photo is uploaded).

### Events Reference

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

## WLED Command Modes

### Preset Mode

Sends `{"preset": N}` to the WLED device topic. Requires you to create presets in WLED first.

**Example**: Preset `4` → `{"preset": 4}`

### Effect Mode

Sends `{"seg": [{"fx": N}]}` where N is the WLED effect ID. No presets needed — effects are built into WLED.

Popular effects for photography studios:

| ID | Name | Description |
|----|------|-------------|
| 1 | Blink | Simple attention grabber |
| 2 | Breathe | Gentle pulsing |
| 3 | Wipe | Directional color wipe |
| 6 | Sweep | Wipe with return |
| 8 | Rainbow | Classic rainbow cycle |
| 41 | Meteor Shower | Shooting star effect |
| 46 | Fire 2012 | Realistic fire simulation |
| 72 | Sunrise | Gradual warm glow |
| 82 | Candy Cane | Red and white running lights |
| 89 | Fireworks 1D | Celebration effect |

See the [full WLED effects list](https://github.com/wled/WLED/wiki/List-of-effects-and-palettes) for all 117+ effects.

### Raw JSON Mode

Sends your custom JSON directly to WLED's API. This gives you full control over every WLED parameter.

**Examples:**

```json
// Set color to red
{"seg": [{"col": [[255, 0, 0]]}]}

// Set brightness to 50%
{"bri": 128}

// Fire effect at half speed
{"seg": [{"fx": 46, "sx": 64}]}

// Blue color with fast transition
{"seg": [{"col": [[0, 0, 255]]}], "transition": 5}

// Complex: rainbow with specific speed and intensity
{"seg": [{"fx": 8, "sx": 128, "ix": 200}]}
```

**Validation**: Raw JSON is validated when you save settings — invalid JSON will be rejected with an error.

## Payload Reference

All payloads are JSON. The structured topic payload includes a `"wled"` field with the resolved command:

```json
{
  "uploadName": "Morning Session",
  "userId": "abc123",
  "wled": {"seg": [{"fx": 46}]}
}
```

Additionally, the resolved command is published directly to `{wledDeviceTopic}/api` for WLED to act on.

## WLED Setup

### Option A: Direct WLED Control (Recommended)

Shutterbase can publish **directly** to your WLED device's API topic — no bridge or automation tool needed.

#### Step-by-Step: Connect WLED

**Step 1: Install and configure MQTT broker**

```bash
# Docker (easiest)
docker run -d --name mosquitto -p 1883:1883 eclipse-mosquitto

# Or install Mosquitto
sudo apt install mosquitto mosquitto-clients
```

Verify broker is running:
```bash
mosquitto_sub -t '$SYS/#' -C 1 -W 2 | grep -q . && echo "Broker running"
```

**Step 2: Configure WLED**

1. Open WLED web UI (e.g. `http://192.168.1.100`)
2. Go to **Config > Sync Interfaces**
3. Scroll to **MQTT** section
4. Enable MQTT
5. Set **Broker** to your broker IP (e.g. `192.168.1.50`)
6. Set **Port** to `1883`
7. Note the **Device Topic** — it looks like `wled/a4cf12fa54b3`
8. Click **Save** and reboot WLED

**Step 3: Test WLED MQTT connection**

Open a terminal and subscribe to WLED's topic:
```bash
# Subscribe to all WLED messages
mosquitto_sub -h 192.168.1.50 -t "wled/#" -v

# In another terminal, send a test command
mosquitto_pub -h 192.168.1.50 -t "wled/a4cf12fa54b3/api" -m '{"on":true}'
```

WLED should turn on. If it works, the MQTT connection is good.

**Step 4: Configure Shutterbase**

1. Go to your project **Settings > General**
2. Scroll to **MQTT / WLED Integration**
3. Fill in:
   - **Broker URL**: `tcp://192.168.1.50:1883` (same broker as WLED)
   - **WLED Device Topic**: `wled/a4cf12fa54b3` (from Step 2)
4. Enable events and choose command modes:
   - **Preset mode**: enter preset numbers (must create presets in WLED first)
   - **Effect mode**: pick from the dropdown (no presets needed)
   - **Raw JSON mode**: enter custom JSON commands
5. Optionally set auto-off durations (in seconds)
6. Click **Save MQTT Settings**

**Step 5: Test the integration**

1. Upload a photo to your project
2. Watch WLED — it should react based on your event settings
3. If nothing happens, check the troubleshooting section below

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

## Examples

### Quick Start: Flash on Approval (Effect Mode)

No presets needed — just pick effects from the dropdown:

1. WLED: note your Device Topic (e.g. `wled/a4cf12fa54b3`)
2. Shutterbase Project Settings > MQTT:
   - Broker: `tcp://mosquitto:1883`
   - WLED Device Topic: `wled/a4cf12fa54b3`
   - **Approved** event: enable, mode = **Effect**, select "Fireworks Starburst" (ID 24)
   - Auto-off: `5` seconds
3. When a reviewer approves an upload, WLED plays Fireworks Starburst for 5 seconds, then turns off

### Preset Mode: Full Review Flow

1. WLED presets:
   - Preset 1: Green pulse (upload created)
   - Preset 3: Blue pulse (ready for review)
   - Preset 4: Gold flash (approved)
   - Preset 5: Red flash (rejected)
2. Shutterbase: enable events with mode = **Preset**, matching preset numbers
3. WLED reacts to each stage of the review process

### Raw JSON Mode: Custom Colors

Set specific colors for each event:

- **Upload created**: `{"seg": [{"col": [[0, 255, 0]]}]}` (green)
- **Approved**: `{"seg": [{"col": [[255, 215, 0]]}]}` (gold)
- **Rejected**: `{"seg": [{"col": [[255, 0, 0]]}]}` (red)
- Auto-off: `3` seconds on all events

### Tag-Based Effects

1. Enable **Tag assigned**, trigger tags: `winner, highlight`
2. Mode = **Effect**, select "Rainbow" (ID 8), auto-off: `10` seconds
3. When a `winner` tag is assigned, WLED shows rainbow for 10 seconds

### Multi-Device (with Bridge)

Different WLED devices react to different events:

- **Office WLED** (`wled/office`): Fireworks Starburst on approvals
- **Studio WLED** (`wled/studio`): Breathe on ready
- **Party WLED** (`wled/party`): Rainbow on tag assigned

Use the **Topic Prefix** for structured topics + Home Assistant/Node-RED to route to each device.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| No messages received | Is the MQTT broker running? Is the Broker URL correct in project settings? |
| WLED not reacting (direct) | Is the WLED Device Topic correct? Is WLED connected to the same broker? |
| WLED not reacting (bridge) | Is the bridge configured? Is Home Assistant/Node-RED subscribed to the right topic? |
| Connection lost warnings | Check broker logs, network connectivity, credentials |
| Events not firing | Is the event toggle enabled in project settings? |
| Tag triggers not working | Is the tag name in the trigger list exactly (case-sensitive)? |
| Raw JSON rejected on save | Is the JSON valid? Check for missing quotes, trailing commas, etc. |
| Effect not visible on WLED | Some effects require 2D matrix or specific LED layouts. Check WLED's effect documentation. |
| Auto-off not working | Is the duration > 0? Check that WLED Device Topic is set (auto-off publishes to the device topic). |

## Using the MQTT Service

### Monitor MQTT Messages

Subscribe to all Shutterbase topics to see what's being published:

```bash
# Subscribe to all Shutterbase messages
mosquitto_sub -h <broker> -t "shutterbase/#" -v

# Subscribe to a specific project
mosquitto_sub -h <broker> -t "shutterbase/<projectId>/#" -v

# Subscribe to a specific upload
mosquitto_sub -h <broker> -t "shutterbase/<projectId>/upload/<uploadId>/#" -v
```

### Test WLED Directly

Send a test command to WLED without Shutterbase:

```bash
# Turn on WLED
mosquitto_pub -h <broker> -t "wled/<deviceId>/api" -m '{"on":true}'

# Set preset 4
mosquitto_pub -h <broker> -t "wled/<deviceId>/api" -m '{"preset":4}'

# Trigger fire effect
mosquitto_pub -h <broker> -t "wled/<deviceId>/api" -m '{"seg":[{"fx":46}]}'

# Set color to red
mosquitto_pub -h <broker> -t "wled/<deviceId>/api" -m '{"seg":[{"col":[[255,0,0]]}]}'

# Set brightness to 128
mosquitto_pub -h <broker> -t "wled/<deviceId>/api" -m '{"bri":128}'
```

### Home Assistant Automation

```yaml
automation:
  - alias: "Shutterbase → WLED"
    trigger:
      - platform: mqtt
        topic: "shutterbase/+/upload/+/approved"
    action:
      - service: mqtt.publish
        data:
          topic: "wled/device1/api"
          payload: '{"seg":[{"fx":24}]}'
```
