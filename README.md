# MongoDB Backup Tool in Go

## mongobackup usage

```
mongobackup
--command <backup, restore, query>
--host <default: localhost>
--port <default:21707>
--database <DB Name>
--user <username>
--password <password>
--backupname <dated backup name for use with restore>
```

A basic Mongo backup tool written in Go.

- When backing up a database it puts the gzip archive into `/tmp`
- The database backups are tracked in the backup.backups collection on the server