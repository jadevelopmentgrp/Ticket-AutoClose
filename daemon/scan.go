package daemon

import (
	"context"
	"github.com/TicketsBot/common/autoclose"
)

func (d *Daemon) scan() (tickets []autoclose.Ticket, err error) {
	query := `
SELECT
	t.guild_id,
	t.id
FROM
	tickets t
INNER JOIN
	auto_close ac
		ON t.guild_id = ac.guild_id
INNER JOIN
	ticket_last_message tlm
		ON t.guild_id = tlm.guild_id AND t.id = tlm.ticket_id
WHERE
	t.open AND
	ac.enabled AND
	(
		ac.since_open_with_no_response > INTERVAL '0 seconds'
		AND
		NOT EXISTS (
			SELECT
			FROM 
				ticket_last_message tlm
			WHERE
				tlm.guild_id = t.guild_id AND 
					tlm.ticket_id = t.id
		)
		AND
		NOW() - t.open_time > ac.since_open_with_no_response
	)
	OR
	(
		ac.since_last_message > INTERVAL '0 seconds'
		AND
		NOW() - tlm.last_message_time > ac.since_last_message
	)
;
`

	// doesn't matter what table we query from, all same conn
	rows, err := d.db.Tickets.Query(context.Background(), query)
	defer rows.Close()

	if err != nil {
		return
	}

	for rows.Next() {
		var ticket autoclose.Ticket
		if err = rows.Scan(&ticket.GuildId, &ticket.TicketId); err != nil {
			return
		}

		tickets = append(tickets, ticket)
	}

	return
}

