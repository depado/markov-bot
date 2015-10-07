# markov-bot

An irc bot that builds markov chains using log files and generate random sentences when called.

(`!markov` to request a random sentance.)

## Status

For now, the bot only parses the `history.log` file. It's in a custom format, so I don't except you will be able to use it.

It is planned to use boltdb but not tonight because I'm quite tired. Also there are multiple bugs I should fix :
 - The bot keeps the `"` char, but should remove it.
 - The bot sometimes add a whitespace at the start of a sentence.
 - The bot should not keep usernames (nicknames) in the markov chain.

Boltdb will be used so that the log file won't be parsed each time the bot starts. Also this bot will be integrated in go-b0tsec as a plugin so at one point, the development on that repository will stop.

Also maybe it won't be a plugin but a middleware that sends random sentence when there is activity on the chan and when it is activated. (It could potentially be highly annoying)

## TODO

 - Fix the above bugs
 - Integrate with boltdb
 - Automatic markov chain generation when someone sends a message
 - Automatic database backup of the markov chain
