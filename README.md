# worker
- worker for webrtc

---
### Architecture

```
----------   sdp/candidate   ----------    sdp/candidate   ----------
| peer A |  -------------->  | signal |   -------------->  | peer B |
|        |  <-------------   | server |   <-------------   |        |
----------    (websocket)    ----------     (websocket)    ----------
   |  |                                                        |  |
   |  |                       media track(rtp)                 |  |
   |  <-------------------------------------------------------->  |
   |                          data channel(sctp)                  |  
   <--------------------------------------------------------------> 

```
---
