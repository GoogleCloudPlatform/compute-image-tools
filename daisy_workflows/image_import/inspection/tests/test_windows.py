import unittest

from boot_inspect.inspectors.os.windows import _from_nt_version
from compute_image_tools_proto import inspect_pb2


class TestNTMapping_Desktop(unittest.TestCase):

  def test_vista(self):
    assert _from_nt_version(
        variant='Client', major_nt=6, minor_nt=0,
        product_name='Windows Vista') == windows('Vista')

  def test_7(self):
    assert _from_nt_version(
        variant='Client', major_nt=6, minor_nt=1,
        product_name='Windows 7 Ultimate') == windows('7')

  def test_8(self):
    assert _from_nt_version(
        variant='Client', major_nt=6, minor_nt=2,
        product_name='Windows 8 Pro') == windows('8')

  def test_8_1(self):
    assert _from_nt_version(
        variant='Client', major_nt=6, minor_nt=3,
        product_name='Windows 8.1 Pro') == windows('8', '1')

  def test_10(self):
    assert _from_nt_version(
        variant='Client', major_nt=10, minor_nt=0,
        product_name='Windows 10 Enterprise') == windows('10')


class TestNTMapping_Server(unittest.TestCase):

  def test_2008(self):
    assert _from_nt_version(
        product_name='Windows Server 2008 Datacenter', variant='Server',
        major_nt=6, minor_nt=0) == windows('2008')

  def test_2008r2(self):
    assert _from_nt_version(
        product_name='Windows Server 2008 R2 Datacenter', variant='Server',
        major_nt=6, minor_nt=1) == windows('2008', 'r2')

  def test_2012(self):
    assert _from_nt_version(
        product_name='Windows Server 2012 Datacenter', variant='Server',
        major_nt=6, minor_nt=2) == windows('2012')

  def test_2012r2(self):
    assert _from_nt_version(
        product_name='Windows Server 2012 R2 Datacenter', variant='Server',
        major_nt=6, minor_nt=3) == windows('2012', 'r2')

  def test_2016(self):
    assert _from_nt_version(
        product_name='Windows Server 2016 Datacenter', variant='Server',
        major_nt=10, minor_nt=0) == windows('2016')

  def test_2019(self):
    assert _from_nt_version(
        product_name='Windows Server 2019 Datacenter', variant='Server',
        major_nt=10, minor_nt=0) == windows('2019')


class TestNTMapping_Unmatched(unittest.TestCase):

  def test_return_None_when_nt10_ambigous(self):
    # NT 10.0 is shared between server 2016 and 2019. Default to
    # None if the product name doesn't include the version.
    assert _from_nt_version(
        product_name='Windows Server', variant='Server',
        major_nt=10, minor_nt=0) is None

  def test_return_None_prior_to_nt_6(self):
    for major in range(1, 6):
      for minor in range(1, 11):
        for variant in ['Server', 'Client']:
          assert _from_nt_version(
              major_nt=major, minor_nt=minor, variant=variant,
              product_name='Windows ' + variant) is None


def windows(major, minor='') -> inspect_pb2.OsRelease:
  return inspect_pb2.OsRelease(
      major_version=major,
      minor_version=minor,
      distro_id=inspect_pb2.Distro.WINDOWS,
  )
