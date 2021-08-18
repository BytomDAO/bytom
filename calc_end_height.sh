# 计算截止块高

# /bin/sh

bytomEndHeight(){
  secondsPerBlock=150
  startTime="2021-08-16 16:39:04"
  endTime="2021-08-20 09:00:00"
  startHeight=707577

  startTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${startTime}" +%s`
  endTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${endTime}" +%s`
  endHeight=`echo "${startHeight} + (${endTimeSec} - ${startTimeSec})/${secondsPerBlock}" | bc`

  echo "bytom current height:" ${startTime} ${startHeight}
  echo "bytom end block height:" ${endTime} ${endHeight}
}

vaporEndHeight(){
  secondsPerBlock=0.5
  startTime="2021-08-16 19:53:03"
  endTime="2021-08-20 08:45:00"
  startHeight=128346718
  startTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${startTime}" +%s`
  endTimeSec=`date -j -f  "%Y-%m-%d %H:%M:%S"  "${endTime}" +%s`
  endHeight=`echo "${startHeight} + (${endTimeSec} - ${startTimeSec})/${secondsPerBlock}" | bc`

  echo "vapor current height:" ${startTime} ${startHeight}
  echo "vapor end block height:" ${endTime} ${endHeight}
}

bytomEndHeight

vaporEndHeight

# 最终确定的高度
# bytom高度：709660
# vapor高度：128957600
