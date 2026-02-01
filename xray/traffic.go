package xray

import "time"

type Traffic struct {
    IsInbound  bool      
    IsOutbound bool      
    Tag        string    
    UserID     int       
    Up         int64     
    Down       int64    
    
   Timestamp  time.Time 
}
