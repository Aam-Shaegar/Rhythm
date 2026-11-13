import unittest
from train import build, classify, features, tokens
class BaselineTests(unittest.TestCase):
 def test_reproducible_and_holdout(self):
  a,metrics=build();b,_=build();self.assertEqual(a,b)
  self.assertGreaterEqual(metrics['category_holdout']['accuracy'],.9)
  self.assertLess(metrics['risk_holdout']['brier'],metrics['risk_holdout']['constant_brier'])
 def test_everyday_wording(self):
  a,_=build()
  for text,expected in [('Сдать лабу по математике','study'),('Купить хлеб и яйца','home'),('Оплатить подписку','finance'),('Записаться к стоматологу','health')]:
   self.assertEqual(classify(a['text'],text)[0],expected,text)
 def test_unknown_and_normalization(self):
  a,_=build();self.assertEqual(classify(a['text'],'zxqv')[0],'other')
  self.assertEqual(tokens('Ёлки!'),tokens('елки'))
  x=features(-100,999999,9999,999,999,True,99999,False,2,True,[0]*6+[1])
  self.assertTrue(all(0<=v<=1 for v in x))
if __name__=='__main__':unittest.main()
