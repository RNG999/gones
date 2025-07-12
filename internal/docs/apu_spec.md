# NES APU (Audio Processing Unit) Specifications

## Overview
The APU is integrated into the 2A03 CPU chip and generates audio output. It contains five sound channels: two pulse wave channels, one triangle wave channel, one noise channel, and one delta modulation channel (DMC).

## Memory Mapped Registers

### Pulse Channel 1 ($4000-$4003)

#### $4000 - Pulse 1 Duty/Envelope
```
7  bit  0
---- ----
DDNE VVVV
|||| ||||
|||| ++++- Volume/Envelope (V)
|||+------ Envelope disable (0: use envelope, 1: use constant volume)
||+------- Length counter halt / Envelope loop
++-------- Duty cycle:
           00: 12.5% (_-------_-------_-------)
           01: 25%  (__------__------__------)
           10: 50%  (____----____----____----)
           11: 75%  (______--______--______--)
```

#### $4001 - Pulse 1 Sweep
```
7  bit  0
---- ----
EPPP NSSS
|||| ||||
|||| |+++- Shift count (pitch bend amount)
|||| +---- Negate (1: pitch up, 0: pitch down)
|+++------ Period (sweep update rate)
+--------- Enable sweep
```

#### $4002 - Pulse 1 Timer Low
```
7  bit  0
---- ----
TTTT TTTT
|||| ||||
++++-++++- Timer low 8 bits
```

#### $4003 - Pulse 1 Timer High/Length
```
7  bit  0
---- ----
LLLL LTTT
|||| ||||
|||| |+++- Timer high 3 bits
++++-+---- Length counter load (5-bit value)
```

### Pulse Channel 2 ($4004-$4007)
Same format as Pulse Channel 1

### Triangle Channel ($4008-$400B)

#### $4008 - Triangle Linear Counter
```
7  bit  0
---- ----
DLLL LLLL
|||| ||||
|+++-++++- Linear counter reload value
+--------- Length counter halt / Linear counter control
```

#### $4009 - Unused

#### $400A - Triangle Timer Low
```
7  bit  0
---- ----
TTTT TTTT
|||| ||||
++++-++++- Timer low 8 bits
```

#### $400B - Triangle Timer High/Length
```
7  bit  0
---- ----
LLLL LTTT
|||| ||||
|||| |+++- Timer high 3 bits
++++-+---- Length counter load
```

### Noise Channel ($400C-$400F)

#### $400C - Noise Envelope
```
7  bit  0
---- ----
--NE VVVV
  || ||||
  || ++++- Volume/Envelope
  |+------ Envelope disable
  +------- Length counter halt / Envelope loop
```

#### $400D - Unused

#### $400E - Noise Period/Mode
```
7  bit  0
---- ----
M--- PPPP
|    ||||
|    ++++- Period index (selects from period table)
+--------- Mode (0: 32k steps, 1: 93 steps)
```

#### $400F - Noise Length
```
7  bit  0
---- ----
LLLL L---
|||| |
++++-+---- Length counter load
```

### DMC Channel ($4010-$4013)

#### $4010 - DMC Flags/Rate
```
7  bit  0
---- ----
IL-- RRRR
||   ||||
||   ++++- Rate index (selects from rate table)
|+-------- Loop (1: loop sample)
+--------- IRQ enable
```

#### $4011 - DMC Direct Load
```
7  bit  0
---- ----
-DDD DDDD
 ||| ||||
 +++-++++- 7-bit DAC value
```

#### $4012 - DMC Sample Address
```
7  bit  0
---- ----
AAAA AAAA
|||| ||||
++++-++++- Sample address = $C000 + (A * 64)
```

#### $4013 - DMC Sample Length
```
7  bit  0
---- ----
LLLL LLLL
|||| ||||
++++-++++- Sample length = (L * 16) + 1 bytes
```

### Control/Status Registers

#### $4015 - Channel Enable/Status (Write)
```
7  bit  0
---- ----
---D NT21
   | ||||
   | |||+- Pulse 1 enable
   | ||+-- Pulse 2 enable
   | |+--- Triangle enable
   | +---- Noise enable
   +------ DMC enable
```

#### $4015 - Channel Status (Read)
```
7  bit  0
---- ----
IF-D NT21
||   ||||
||   |||+- Pulse 1 length counter > 0
||   ||+-- Pulse 2 length counter > 0
||   |+--- Triangle length counter > 0
||   +---- Noise length counter > 0
|+-------- DMC bytes remaining > 0
+--------- DMC interrupt flag
```

#### $4017 - Frame Counter
```
7  bit  0
---- ----
MI-- ----
||
|+-------- Interrupt inhibit
+--------- Mode (0: 4-step, 1: 5-step)
```

## Audio Generation

### Pulse Channels
- Generate square waves with variable duty cycle
- 11-bit timer controls frequency
- Frequency = CPU_freq / (16 * (timer + 1))
- Sweep unit can automatically bend pitch
- Output silenced if timer < 8

### Triangle Channel
- Generates triangle wave (no volume control)
- Linear counter provides additional duration control
- Frequency = CPU_freq / (32 * (timer + 1))
- Ultrasonic frequencies (timer < 2) output silence

### Noise Channel
- Generates pseudo-random noise using 15-bit LFSR
- Two modes: long (32767 steps) and short (93 steps)
- Period table selects from 16 preset rates

### DMC Channel
- Plays 7-bit delta-encoded samples
- Samples stored in PRG-ROM at $C000-$FFEA
- Can loop samples continuously
- Steals CPU cycles for sample fetches

## Length Counter

Length counter values (loaded into bits 7-3 of $4003/$4007/$400B/$400F):

| Value | Length | Value | Length |
|-------|--------|-------|--------|
| $00 | 10 | $10 | 12 |
| $01 | 254 | $11 | 16 |
| $02 | 20 | $12 | 24 |
| $03 | 2 | $13 | 8 |
| $04 | 40 | $14 | 48 |
| $05 | 4 | $15 | 6 |
| $06 | 80 | $16 | 96 |
| $07 | 6 | $17 | 4 |
| $08 | 160 | $18 | 192 |
| $09 | 8 | $19 | 2 |
| $0A | 60 | $1A | 72 |
| $0B | 10 | $1B | 16 |
| $0C | 14 | $1C | 28 |
| $0D | 12 | $1D | 32 |
| $0E | 26 | $1E | 52 |
| $0F | 14 | $1F | 2 |

## Frame Counter

### 4-Step Mode (Mode = 0)
```
Step   Cycles    Envelope/Linear    Length/Sweep
0      7457      Clock              -
1      14913     Clock              Clock
2      22371     Clock              -
3      29829     Clock              Clock
-      29830     IRQ (if enabled)   -
```

### 5-Step Mode (Mode = 1)
```
Step   Cycles    Envelope/Linear    Length/Sweep
0      7457      Clock              -
1      14913     Clock              Clock
2      22371     Clock              -
3      29829     -                  -
4      37281     Clock              Clock
```

## DMC Rate Table

| Index | NTSC Rate | PAL Rate |
|-------|-----------|----------|
| $0 | 428 | 398 |
| $1 | 380 | 354 |
| $2 | 340 | 316 |
| $3 | 320 | 298 |
| $4 | 286 | 276 |
| $5 | 254 | 236 |
| $6 | 226 | 210 |
| $7 | 214 | 198 |
| $8 | 190 | 176 |
| $9 | 160 | 148 |
| $A | 142 | 132 |
| $B | 128 | 118 |
| $C | 106 | 98 |
| $D | 84 | 78 |
| $E | 72 | 66 |
| $F | 54 | 50 |

## Noise Period Table

| Index | NTSC Period | PAL Period |
|-------|-------------|------------|
| $0 | 4 | 4 |
| $1 | 8 | 8 |
| $2 | 16 | 14 |
| $3 | 32 | 30 |
| $4 | 64 | 60 |
| $5 | 96 | 88 |
| $6 | 128 | 118 |
| $7 | 160 | 148 |
| $8 | 202 | 188 |
| $9 | 254 | 236 |
| $A | 380 | 354 |
| $B | 508 | 472 |
| $C | 762 | 708 |
| $D | 1016 | 944 |
| $E | 2034 | 1890 |
| $F | 4068 | 3778 |

## Mixer

The APU mixes channels using non-linear mixing:

### Pulse Mixing
```
pulse_out = 95.88 / ((8128 / (pulse1 + pulse2)) + 100)
```

### Other Channel Mixing
```
tnd_out = 159.79 / (1 / ((triangle / 8227) + (noise / 12241) + (dmc / 22638)) + 100)
```

### Final Output
```
output = pulse_out + tnd_out
```

## Important Implementation Notes

1. **Sweep Unit Behavior**: The sweep unit on pulse channel 1 uses different subtraction behavior than pulse channel 2, creating slightly different sweep ranges.

2. **Phase Reset**: Writing to $4003/$4007 resets the pulse phase and envelope. Writing to $400B doesn't reset the triangle phase.

3. **DMC Conflicts**: DMC sample fetches can conflict with CPU reads, potentially corrupting data. Games work around this by avoiding sensitive operations during sample playback.

4. **Frame Counter Quirks**: Writing to $4017 resets the frame counter after 3-4 CPU cycles. The exact delay depends on whether the write occurs on an even or odd CPU cycle.

5. **Length Counter Clocking**: The length counter is clocked only when the channel is enabled. Disabling a channel doesn't clear its length counter.

6. **Triangle Linear Counter**: The linear counter reload flag is set whenever $4008 is written or when the length counter is loaded by writing to $400B.

7. **DMC IRQ**: The DMC IRQ flag is set when the sample buffer empties, not when the sample finishes playing. This allows the CPU to keep the DMC fed with minimal gaps.