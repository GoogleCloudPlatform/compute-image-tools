import calendar
import datetime


def utc_timestamp():
    return calendar.timegm(datetime.datetime.utcnow().utctimetuple())
