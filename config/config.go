package config

import "time"

var DETECTION_PORT string = ":14272"
var TCP_LISTEN_PORT string = ":14279"
var HALL_LIGHTS_PORT string = ":14274"
var TCP_BACKUP_PORT string = ":14275"
var TCP_NEW_PRIMARY_LISTEN_PORT string = ":14276"
var LOCAL_ELEVATOR_ALIVE_PORT string = ":17878"

const BUFSIZE = 1024

const UDPINTERVAL = 2 * time.Second
const TIMEMOUT = 500 * time.Millisecond
