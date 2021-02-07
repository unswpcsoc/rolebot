package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const (
	MAGIC  = "💻"
	MEMBER = "member"
)

var (
	unregister func()
	errs       = log.New(os.Stderr, "Error: ", log.Ltime) // logger for errors
	roles      = map[string]string{
		"🗾": "weeb",
		"🤔": "meta",
		"🧾": "bookworm",
	}
)

func main() {
	// discordgo init
	key, ok := os.LookupEnv("KEY")
	if !ok {
		errs.Fatalln("Missing Discord API Key: Set env var $KEY")
	}

	dgo, err := discordgo.New("Bot " + key)
	if err != nil {
		errs.Fatalln(err)
	}

	err = dgo.Open()
	if err != nil {
		errs.Fatalln(err)
	}

	log.Printf("Logged in as: %v", dgo.State.User.ID)
	defer dgo.Close()

	// set status
	dgo.UpdateListeningStatus("the mods 🥵")

	// ensure all roles actually exist and remap
	guilds, err := dgo.UserGuilds(1, "", "")
	if err != nil {
		errs.Fatalln(err)
	}

	thisGuild := guilds[0].ID

	guildroles, err := dgo.GuildRoles(thisGuild)
	if err != nil {
		errs.Fatalln(err)
	}

	roleIDs := map[string]string{}
	for _, role := range roles {
		sentinel := false
		for _, gr := range guildroles {
			if role == strings.ToLower(gr.Name) {
				roleIDs[role] = gr.ID
				sentinel = true
				break
			}
		}

		if !sentinel {
			errs.Fatalf("role %s is not in guild %s", role, thisGuild)
		}
	}

	// initialise magic react handler
	unregister = dgo.AddHandler(func(ses *discordgo.Session, mer *discordgo.MessageReactionAdd) {
		// check for magic
		if mer.Emoji.Name == MAGIC {
			// TODO: when magic present, register the reaction handler
			log.Println("registering add handler")
			ses.AddHandler(func(s *discordgo.Session, mr *discordgo.MessageReactionAdd) {
				// guard unverified members
				mem, err := s.GuildMember(mr.GuildID, mr.UserID)
				if err != nil {
					errs.Println(err)
				}

				isVerified := false
				for _, role := range mem.Roles {
					if strings.ToLower(role) == MEMBER {
						isVerified = true
					}
				}

				if !isVerified {
					log.Printf("Unverified member %s attempted to gain %s", mr.UserID, mr.Emoji.Name)
					return
				}

				for emoji, role := range roles {
					if emoji == mr.Emoji.Name {
						// assign role
						err = s.GuildMemberRoleAdd(thisGuild, mr.UserID, roleIDs[role])
						if err != nil {
							errs.Println(err)
						}
					}
				}
			})

			// register
			log.Println("registering remove handler")
			ses.AddHandler(func(s *discordgo.Session, mr *discordgo.MessageReactionRemove) {
				// guard unverified members
				mem, err := s.GuildMember(mr.GuildID, mr.UserID)
				if err != nil {
					errs.Println(err)
				}

				isVerified := false
				for _, role := range mem.Roles {
					if strings.ToLower(role) == MEMBER {
						isVerified = true
					}
				}

				if !isVerified {
					log.Printf("Unverified member %s attempted to gain %s", mr.UserID, mr.Emoji.Name)
					return
				}

				for emoji, role := range roles {
					if emoji == mr.Emoji.Name {
						// assign role
						err = s.GuildMemberRoleRemove(thisGuild, mr.UserID, roleIDs[role])
						if err != nil {
							errs.Println(err)
						}
					}
				}
			})

			// react to the message with each emoji
			for emoji, _ := range roles {
				ses.MessageReactionAdd(mer.ChannelID, mer.MessageID, emoji)
			}
			unregister()
		}
	})

	// keep alive
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	sig := <-sc

	log.Println("Received Signal: " + sig.String())
	log.Println("Bye!")
}
