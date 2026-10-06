# Operator customization. Never emit a welcome banner into SSH command/SCP/SFTP streams.
case $- in
	*i*) /usr/local/libexec/soda/soda-console-welcome ;;
esac
