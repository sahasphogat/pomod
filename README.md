history is stored in ~/.local/share/pomod/log.jsonl

## hooks
hooks can be used to run commands between sessions  
these are the possible hooks make them in ~/.local/share/pomod/hooks/  
break_finished     paused      session_finished  
break_started      resumed     session_started

# pomod
run it as daemon

# pomod-client
toggle - pause/resume (runs hook paused/resumed)  
finish - finishes current session and skips to next  
status - shows seconds, running status, and mode  

