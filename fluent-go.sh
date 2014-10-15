#!/bin/sh

### BEGIN INIT INFO
# Provides:          fluent-go
# Required-Start:    $local_fs $remote_fs $network $syslog
# Required-Stop:     $local_fs $remote_fs $network $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: starts the fluent-go
# Description:       starts fluent-go using start-stop-daemon
### END INIT INFO

PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
DIR=/vagrant/fluent-go
DAEMON=$DIR/fluent-go
LOCK_FILE="$DIR/auto_restart_when_stop"
NAME=fluent-go
DESC=fluent-go

test -x $DAEMON || exit 255

set -e

. /lib/lsb/init-functions

case "$1" in
  start)
        echo -n "Starting $DESC: "
        start-stop-daemon --start --oknodo --background --quiet -m --pidfile $DIR/$NAME.pid \
          --exec $DAEMON -d $DIR -- $DAEMON_OPTS || true
        echo "$NAME."
                touch $LOCK_FILE
        ;;
  stop)
        echo -n "Stopping $DESC: "
        start-stop-daemon --stop --quiet --pidfile $DIR/$NAME.pid \
                --exec $DAEMON -d $DIR || true
        echo "$NAME."
                rm -f $LOCK_FILE
        ;;
  restart|force-reload)
        echo -n "Restarting $DESC: "
        start-stop-daemon --stop --quiet --pidfile \
                $DIR/$NAME.pid --exec $DAEMON -d $DIR || true
        sleep 1
        start-stop-daemon --start --oknodo --background --quiet -m --pidfile \
                $DIR/$NAME.pid --exec $DAEMON -d $DIR -- $DAEMON_OPTS || true
                touch $LOCK_FILE
        echo "$NAME."
        ;;
  status)
        status_of_proc -p $DIR/$NAME.pid "$DAEMON" $NAME && exit 0 || exit $?
        ;;
  *)
        echo "Usage: $NAME {start|stop|restart|force-reload|status}" >&2
        exit 1
        ;;
esac

exit 0
