#usage: <script name> [setpoint]

wait_until_run_busy_cleared() {
  run_busy=1
  while [[ $run_busy -ne 0 ]]
  do
    rd_interface=`rdmsr -p $core_id 0xb0`
    run_busy=$[rd_interface & 0x80000000]
    if [ $run_busy -eq 0 ]; then
      #not busy, just return
      break
    else
      echo "====warning:RUN_BUSY=1.sleep 1,then retry"
      sleep 1
    fi
  done
}

disable_hwdrc() {
  # disable HWDRC
  wrmsr -p $core_id 0xb1 0x0
  wrmsr -p $core_id 0xb0 0x810054d0
  wait_until_run_busy_cleared

  echo "disable_hwdrc: $core_id"
}

enable_hwdrc() {
  # enable HWDRC
  wrmsr -p $core_id 0xb1 0x2
  wrmsr -p $core_id 0xb0 0x810054d0
  wait_until_run_busy_cleared

  echo "enable_hwdrc: $core_id"
}

config_hwdrc() {
  # map CLOSes & MCLOSes (low_closids to memCLOS 1(LP), all others to memCLOS 0(HP))
  echo "Low CLOSids: $1"

  low_closes=$1
  array=(${low_closes//,/ })

  map=0
  for low_closid in ${array[@]}
  do
	map=$(($map + $((1 << 2*$low_closid))))
  done

  echo "maps: " 
  echo "obase=2; $map" | bc

  wrmsr -p $core_id 0xb1 $map
  wrmsr -p $core_id 0xb0 0x810050d0
  wait_until_run_busy_cleared

  # memCLOS 0(HP) with MAX delay 0x1, MIN delay 0x1, priority 0x0
  wrmsr -p $core_id 0xb1 0x80010100
  wrmsr -p $core_id 0xb0 0x810051d0
  wait_until_run_busy_cleared

  # memCLOS 1(LP) with MAX delay 0xff, MIN delay 0x1, priority 0xF
  wrmsr -p $core_id 0xb1 0x81ff010f
  wrmsr -p $core_id 0xb0 0x810851d0
  wait_until_run_busy_cleared

  # memCLOS 2 with MAX delay 0xff, MIN delay 0x1, priority 0x5
  wrmsr -p $core_id 0xb1 0x82ff0105
  wrmsr -p $core_id 0xb0 0x811051d0
  wait_until_run_busy_cleared

  # memCLOS 3 with MAX delay 0xff, MIN delay 0x1, priority 0xA
  wrmsr -p $core_id 0xb1 0x83ff010a
  wrmsr -p $core_id 0xb0 0x811851d0
  wait_until_run_busy_cleared

  #MEM_CLOS_EVENT=0x80 MCLOS_RPQ_OCCUPANCY_EVENT
  #MEM_CLOS_TIME_WINDOW=0x01
  #MEMCLOS_SET_POINT=0x01
  wrmsr -p $core_id 0xb1 0x01800101
  wrmsr -p $core_id 0xb0 0x810052d0
  wait_until_run_busy_cleared
}

execute() {
  cpus=$(lscpu |grep "NUMA node" | awk '{print $4}')
  for c in ${cpus[@]}
  do
    core_id=${c%%-*}
    $1
  done
}

if [ ! -e /sys/fs/resctrl/c1/closid ] || [ ! -e /sys/fs/resctrl/c2/closid ]; then
  echo "error: resctrl group doesn't exist!"
  exit
fi

high_closid=$(cat /sys/fs/resctrl/c1/closid)
low_closid=$(cat /sys/fs/resctrl/c2/closid)

echo "high_closid: $high_closid, low_closid: $low_closid"

#execute disable_hwdrc
#execute config_hwdrc
#execute enable_hwdrc
