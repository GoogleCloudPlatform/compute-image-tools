import calendar
import datetime


def urljoin(*args):
    return '/'.join(args)


def utc_timestamp():
    return calendar.timegm(datetime.datetime.utcnow().utctimetuple())
