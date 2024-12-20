# discordutils

A collection of command-line utilities for Discord.

## Installation
Requires Go 1.16+.

```bash
go install samhza.com/discordutils/cmd/...@latest         # install all tools
go install samhza.com/discordutils/cmd/deferreddel@latest # install a single tool
```

## Authentication
All tools will look for a Discord token in `~/.config/discord-token` if not provided via the `-tok` flag. To extract and save your token:

```bash
# Linux
discordtok > ~/.config/discord-token

# macOS
discordtok > ~/Library/Application\ Support/discord-token

# Windows
discordtok > %APPDATA%\discord-token
```

## deferreddel
Automatically deletes your messages after a specified duration.
You can specify guilds using the `-g` flag, and if you don't specify any guilds then all of your messages
will self-destruct.

```
Usage:
  -dur duration
    	delay for deleting messages (default 48h0m0s)
  -g value
    	guild ID to delete messages from (can be specified multiple times)
  -tok string
    	token
  -v	log queued message deletions
```

## discordtok
Extracts Discord authentication token from your local Discord installation (supports regular, Canary, and PTB versions).

## dsendto
Uploads files to Discord channels.

```
Usage:
  -ch value
    	channel ID
  -f string
    	input file (default "-")
  -n string
    	file name (default "stdout.txt")
  -tok string
    	token
```

## snowstamp
Converts Discord snowflake IDs to human-readable timestamps.

```
Usage:
  snowstamp <snowflake> [snowflake...]
```

