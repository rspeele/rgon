package rcon

type ConsoleCmd struct {
	Var string
	Arg string
}

/* Split a line, e.g. "say \"hello\";rcon_password farm"
 * into its invoked commands.
 */

func ParseConsoleCommands(input string) []ConsoleCmd {
	var cmds []ConsoleCmd
	var vpart, apart string
	var laststmt int
	quoted := false
	input += ";"
	for i := 0; i < len(input); i++ {
		switch c := input[i]; {
		case c == '"':
			quoted = !quoted
		case quoted:
			apart += string(c)
		case (c == ' ' || c == '\t') && apart == "":
			if vpart == "" {
				vpart = input[laststmt:i]
			}
		case c == ';':
			if vpart == "" {
				vpart = input[laststmt:i]
			}
			if vpart != "" {
				cmds = append(cmds, ConsoleCmd{vpart, apart})
			}
			vpart, apart = "", ""
			laststmt = i + 1
		case vpart != "":
			apart += string(c)
		}
	}
	return cmds
}
