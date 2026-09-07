# Discord Server Membership Rules

These rules check membership of a Discord **server (Guild)** and the elapsed
time since the member's `joined_at`. They do not measure account registration
age, binding age, or time in a text/voice channel.

## Setup

1. In Admin > Providers > Discord, retain `identify` and `email` and add
   `guilds.members.read` to Scopes. The `guilds` scope is optional for targeted
   member queries.
2. Enter the servers to check in **Guild IDs**. Separate IDs with spaces or
   commas. IDs are strings; no server ID is hardcoded. An empty list disables
   membership queries.
3. Save and have existing users authorize Discord again to grant any newly
   added scope. Existing tokens are not upgraded by changing configuration.
4. In Admin > Security Rules, select **Discord server membership** or
   **Discord server membership age** and enter the same Guild ID.

For static configuration, `oauth2.discord.guild_ids` accepts a YAML list of
quoted IDs. As with other provider settings, database/admin configuration
takes precedence over static configuration.

## Rule Example

Replace the example ID below with your Guild ID. Import this object using the
rule import dialog. Membership age already requires current membership, so a
separate membership condition is not necessary.

```json
{
  "name": "Discord member for more than 60 days",
  "level": 2,
  "priority": 0,
  "is_active": true,
  "conditions": {
    "operator": "AND",
    "items": [
      {
        "condition": {
          "type": "discord_guild_age_days",
          "field": "123456789012345678",
          "operator": "gt",
          "min_days": 60
        }
      }
    ]
  }
}
```

For membership alone, use `discord_guild_member`, the same `field` (Guild ID),
`operator: "eq"`, and `value: true`. `value: false` requires an explicitly
confirmed non-member result; missing data or a permission/API error never
matches either value.

`gte` means at least the specified duration; `gt` means strictly longer.
Days are 24-hour periods, compared without rounding down. 60 days is not
necessarily two calendar months. The timestamp is Discord's current member
`joined_at`; no historical memberships are accumulated.

## Updates and Failures

Membership is refreshed during Discord authorization and by the existing
`social_auth_sync` background job (enabled by default, hourly by default).
The job also recalculates the user's trust level after refreshing Discord.
Leaving a server is reflected on the next successful check, not instantly.
The Recompute button evaluates stored snapshots; it does not contact Discord.

Only configured Guilds are requested. Each request has a 3-second timeout,
with an 8-second budget for the whole group of member requests. Rate limits,
timeouts, missing scopes, and malformed responses replace old membership
data with an unknown result, so membership rules do not pass on stale data
after a failed refresh. Basic Discord authentication can still succeed.

Snapshots are stored in the existing `social_bindings.raw_profile` JSON under
`guilds.<GUILD_ID>.is_member`, `joined_at`, and `checked_at`. No database
migration or bot token is needed.

Rules still calculate a trust level using the first matching rule, ordered
by level and priority. To gate a downstream app, set its minimum trust level
accordingly and ensure other rules at that level or higher do not grant
access through an unintended alternative.

API reference: [Get Current User Guild Member](https://docs.discord.com/developers/resources/user#get-current-user-guild-member).
