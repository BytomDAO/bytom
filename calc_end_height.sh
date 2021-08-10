# 计算截止块高

# /bin/sh

bytomEndHeight(){
  secondsPerBlock=150
  startTime="2021-08-10 17:30:39"
  endTime="2021-08-11 10:00:00"
  startHeight=703961
  startTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${startTime}" +%s`
  endTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${endTime}" +%s`
  endHeight=`echo "${startHeight} + (${endTimeSec} - ${startTimeSec})/${secondsPerBlock}" | bc`

  echo "bytom end block height:" ${endHeight}
}

vaporEndHeight(){
  secondsPerBlock=0.5
  startTime="2021-08-10 17:53:41"
  endTime="2021-08-11 10:00:00"
  startHeight=127315855
  startTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${startTime}" +%s`
  endTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${endTime}" +%s`
  endHeight=`echo "${startHeight} + (${endTimeSec} - ${startTimeSec})/${secondsPerBlock}" | bc`

  echo "vapor end block height:" ${endHeight}
}

bytomEndHeight

vaporEndHeight
