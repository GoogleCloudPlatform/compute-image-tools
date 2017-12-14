import calendar
import datetime


def urljoin(*args):
    return '/'.join(args)


def unix_time():
    return calendar.timegm(datetime.datetime.utcnow().utctimetuple())
