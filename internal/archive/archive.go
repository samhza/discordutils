package archive

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS Message (
	id INTEGER NOT NULL PRIMARY KEY,
	author INTEGER NOT NULL,
	channel INTEGER NOT NULL,
	guild INTEGER,
	content TEXT NOT NULL,
	json TEXT NOT NULL
);
`

var stmtInsert *sql.Stmt

func NewOutput(dir string) (*Output, error) {
	o := new(Output)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return nil, err
	}
	o.DB, err = sql.Open("sqlite3", path.Join(dir, "messages.db"))
	if err != nil {
		return nil, err
	}
	_, err = o.Exec(schema)
	if err != nil {
		return nil, err
	}
	stmtInsert, err = o.Prepare("INSERT INTO Message (id, author, channel, guild, content, json) VALUES(?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}
	o.attdir = path.Join(dir, "attachments")
	return o, nil
}

type Output struct {
	*sql.DB
	attdir string
}

func (o *Output) LogMessage(m discord.Message) error {
	var guild string
	if m.GuildID.IsNull() {
		guild = "dm"
	} else {
		guild = m.GuildID.String()
	}
	attd := path.Join(o.attdir, guild, m.ChannelID.String())
	err := os.MkdirAll(attd, 0777)
	if err != nil {
		return err
	}
	for n, att := range m.Attachments {
		attf := path.Join(attd, fmt.Sprintf("%d,%d %s",
			m.ID,
			n,
			att.Filename,
		))
		f, err := os.Create(attf)
		if err != nil {
			return fmt.Errorf("creating attachment file: %w", err)
		}
		resp, err := http.Get(att.URL)
		if err != nil {
			f.Close()
			return fmt.Errorf("requesting attachment contents: %w", err)
		}
		_, err = io.Copy(f, resp.Body)
		f.Close()
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("downloading attachment: %w", err)
		}
	}
	content := m.Content
	m.Content = ""
	j, err := json.Marshal(m)
	if _, err := stmtInsert.Exec(m.ID, m.Author.ID, m.ChannelID, m.GuildID, content, j); err != nil {
		if e, ok := err.(sqlite3.Error); !ok || e.Code != sqlite3.ErrConstraint {
			return err
		}
	}
	return nil
}
